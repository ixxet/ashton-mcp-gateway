package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type OccupancyReader interface {
	CurrentOccupancy(ctx context.Context, tool manifest.Tool, facilityID string) (athena.Occupancy, error)
}

type Service struct {
	registry manifest.Registry
	athena   OccupancyReader
	logger   *slog.Logger
}

type ToolCallResult struct {
	FacilityID    string `json:"facility_id"`
	CurrentCount  int    `json:"current_count"`
	ObservedAt    string `json:"observed_at"`
	SourceService string `json:"source_service"`
	LatencyMS     int64  `json:"latency_ms"`
}

type ToolCallError struct {
	StatusCode int
	Message    string
}

func (e *ToolCallError) Error() string {
	return e.Message
}

func NewService(registry manifest.Registry, athenaClient OccupancyReader, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		registry: registry,
		athena:   athenaClient,
		logger:   logger,
	}
}

func (s *Service) CallTool(ctx context.Context, name string, arguments map[string]any) (ToolCallResult, error) {
	startedAt := time.Now()

	tool, ok := s.registry.Tool(name)
	if !ok {
		s.logCall(name, "", "", 0, "unknown_tool")
		return ToolCallResult{}, &ToolCallError{
			StatusCode: 404,
			Message:    fmt.Sprintf("unknown tool %q", name),
		}
	}

	facilityID, err := requiredString(arguments, "facility_id")
	if err != nil {
		s.logCall(tool.Name, tool.Upstream.Service, "", 0, "invalid_arguments")
		return ToolCallResult{}, &ToolCallError{
			StatusCode: 400,
			Message:    err.Error(),
		}
	}

	occupancy, err := s.athena.CurrentOccupancy(ctx, tool, facilityID)
	latencyMS := time.Since(startedAt).Milliseconds()
	if err != nil {
		s.logCall(tool.Name, tool.Upstream.Service, facilityID, latencyMS, "upstream_error")
		return ToolCallResult{}, &ToolCallError{
			StatusCode: 502,
			Message:    err.Error(),
		}
	}

	s.logCall(tool.Name, occupancy.Source, facilityID, latencyMS, "success")
	return ToolCallResult{
		FacilityID:    occupancy.FacilityID,
		CurrentCount:  occupancy.CurrentCount,
		ObservedAt:    occupancy.ObservedAt,
		SourceService: occupancy.Source,
		LatencyMS:     latencyMS,
	}, nil
}

func (s *Service) logCall(toolName, sourceService, facilityID string, latencyMS int64, outcome string) {
	s.logger.Info(
		"gateway tool call",
		"tool_name", toolName,
		"source_service", sourceService,
		"facility_id", facilityID,
		"latency_ms", latencyMS,
		"outcome", outcome,
	)
}

func requiredString(arguments map[string]any, key string) (string, error) {
	value, ok := arguments[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}

	text, ok := value.(string)
	if !ok || text == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}

	return text, nil
}
