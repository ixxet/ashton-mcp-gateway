package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/audit"
	"github.com/ixxet/ashton-mcp-gateway/internal/identity"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type OccupancyReader interface {
	CurrentOccupancy(ctx context.Context, tool manifest.Tool, arguments map[string]string) (athena.Occupancy, error)
}

type Service struct {
	registry manifest.Registry
	athena   OccupancyReader
	audit    audit.Recorder
	logger   *slog.Logger
}

type ToolCallResult struct {
	FacilityID    string `json:"facility_id"`
	ZoneID        string `json:"zone_id,omitempty"`
	CurrentCount  int    `json:"current_count"`
	ObservedAt    string `json:"observed_at"`
	SourceService string `json:"source_service"`
	LatencyMS     int64  `json:"latency_ms"`
}

type ToolCallError struct {
	StatusCode int
	Message    string
	Kind       string
}

func (e *ToolCallError) Error() string {
	return e.Message
}

func NewService(registry manifest.Registry, athenaClient OccupancyReader, auditRecorder audit.Recorder, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		registry: registry,
		athena:   athenaClient,
		audit:    auditRecorder,
		logger:   logger,
	}
}

func (s *Service) CallTool(ctx context.Context, caller identity.Caller, name string, arguments map[string]any) (ToolCallResult, error) {
	startedAt := time.Now()
	sanitizedArguments := sanitizeArguments(arguments)

	tool, ok := s.registry.Tool(name)
	if !ok {
		callErr := &ToolCallError{
			StatusCode: 404,
			Message:    fmt.Sprintf("unknown tool %q", name),
			Kind:       "unknown_tool",
		}
		s.logCall(name, "", arguments, 0, callErr.Kind)
		return ToolCallResult{}, s.finalizeCall(ctx, caller, name, "", sanitizedArguments, ToolCallResult{}, callErr, startedAt)
	}

	requiredArguments, err := requiredArguments(arguments, tool.Input.Required)
	if err != nil {
		callErr := &ToolCallError{
			StatusCode: 400,
			Message:    err.Error(),
			Kind:       "invalid_arguments",
		}
		s.logCall(tool.Name, tool.Upstream.Service, arguments, 0, callErr.Kind)
		return ToolCallResult{}, s.finalizeCall(ctx, caller, tool.Name, tool.Upstream.Service, sanitizedArguments, ToolCallResult{}, callErr, startedAt)
	}

	occupancy, err := s.athena.CurrentOccupancy(ctx, tool, requiredArguments)
	latencyMS := time.Since(startedAt).Milliseconds()
	if err != nil {
		callErr := &ToolCallError{
			StatusCode: 502,
			Message:    err.Error(),
			Kind:       "upstream_error",
		}
		s.logCall(tool.Name, tool.Upstream.Service, arguments, latencyMS, callErr.Kind)
		return ToolCallResult{}, s.finalizeCall(ctx, caller, tool.Name, tool.Upstream.Service, sanitizedArguments, ToolCallResult{}, callErr, startedAt)
	}

	result := ToolCallResult{
		FacilityID:    occupancy.FacilityID,
		ZoneID:        occupancy.ZoneID,
		CurrentCount:  occupancy.CurrentCount,
		ObservedAt:    occupancy.ObservedAt,
		SourceService: occupancy.Source,
		LatencyMS:     latencyMS,
	}
	s.logCall(tool.Name, occupancy.Source, arguments, latencyMS, "success")
	return result, s.finalizeCall(ctx, caller, tool.Name, occupancy.Source, sanitizedArguments, result, nil, startedAt)
}

func (s *Service) finalizeCall(ctx context.Context, caller identity.Caller, toolName, sourceService, sanitizedArguments string, result ToolCallResult, callErr *ToolCallError, startedAt time.Time) error {
	latencyMS := time.Since(startedAt).Milliseconds()
	statusCode := 200
	outcome := "success"
	errorKind := ""
	if callErr != nil {
		statusCode = callErr.StatusCode
		outcome = callErr.Kind
		errorKind = callErr.Kind
	}

	resultSummary, err := summarizeResult(result)
	if err != nil {
		return &ToolCallError{
			StatusCode: 500,
			Message:    "audit result summary failed",
			Kind:       "audit_failure",
		}
	}

	if err := s.audit.Record(ctx, audit.Entry{
		OccurredAt:             time.Now().UTC(),
		CallerType:             caller.Type,
		CallerID:               caller.ID,
		CallerDisplay:          caller.Display,
		ToolName:               toolName,
		SourceService:          sourceService,
		SanitizedArgumentsJSON: sanitizedArguments,
		Outcome:                outcome,
		HTTPStatus:             statusCode,
		LatencyMS:              latencyMS,
		ErrorKind:              errorKind,
		ResultSummaryJSON:      resultSummary,
	}); err != nil {
		s.logger.Error("gateway audit persistence failed", "tool_name", toolName, "error", err)
		return &ToolCallError{
			StatusCode: 500,
			Message:    "persisted audit write failed",
			Kind:       "audit_failure",
		}
	}

	if callErr == nil {
		return nil
	}

	return callErr
}

func (s *Service) logCall(toolName, sourceService string, arguments map[string]any, latencyMS int64, outcome string) {
	attrs := []any{
		"tool_name", toolName,
		"source_service", sourceService,
		"latency_ms", latencyMS,
		"outcome", outcome,
	}
	keys := slices.Sorted(maps.Keys(arguments))
	for _, key := range keys {
		value, ok := arguments[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok || text == "" {
			continue
		}
		attrs = append(attrs, key, text)
	}
	s.logger.Info(
		"gateway tool call",
		attrs...,
	)
}

func requiredArguments(arguments map[string]any, required []string) (map[string]string, error) {
	values := make(map[string]string, len(required))
	for _, key := range required {
		value, ok := arguments[key]
		if !ok {
			return nil, fmt.Errorf("%s is required", key)
		}

		text, ok := value.(string)
		if !ok || text == "" {
			return nil, fmt.Errorf("%s must be a non-empty string", key)
		}
		values[key] = text
	}
	return values, nil
}

func sanitizeArguments(arguments map[string]any) string {
	payload := make(map[string]any, len(arguments))
	keys := slices.Sorted(maps.Keys(arguments))
	for _, key := range keys {
		value, ok := arguments[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			payload[key] = truncateString(typed, 256)
		case bool, float64, int, int64, nil:
			payload[key] = typed
		default:
			payload[key] = "[redacted]"
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func summarizeResult(result ToolCallResult) (string, error) {
	if result.FacilityID == "" && result.CurrentCount == 0 && result.ObservedAt == "" && result.SourceService == "" && result.ZoneID == "" {
		return "{}", nil
	}

	encoded, err := json.Marshal(map[string]any{
		"facility_id":    result.FacilityID,
		"zone_id":        result.ZoneID,
		"current_count":  result.CurrentCount,
		"observed_at":    result.ObservedAt,
		"source_service": result.SourceService,
	})
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
