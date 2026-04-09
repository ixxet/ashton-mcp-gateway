package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	OccurredAt             time.Time
	CallerType             string
	CallerID               string
	CallerDisplay          string
	ToolName               string
	SourceService          string
	SanitizedArgumentsJSON string
	Outcome                string
	HTTPStatus             int
	LatencyMS              int64
	ErrorKind              string
	ResultSummaryJSON      string
}

type Recorder interface {
	Record(ctx context.Context, entry Entry) error
}

type Store struct {
	db *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open audit database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping audit database: %w", err)
	}

	store := &Store{db: db}
	if err := store.EnsureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() {
	if s == nil || s.db == nil {
		return
	}
	s.db.Close()
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS gateway_audit_log (
    audit_id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL,
    caller_type TEXT NOT NULL,
    caller_id TEXT NOT NULL,
    caller_display TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    source_service TEXT NOT NULL,
    sanitized_arguments_json JSONB NOT NULL,
    outcome TEXT NOT NULL,
    http_status INTEGER NOT NULL,
    latency_ms BIGINT NOT NULL,
    error_kind TEXT NOT NULL,
    result_summary_json JSONB NOT NULL
)`); err != nil {
		return fmt.Errorf("ensure audit schema: %w", err)
	}
	return nil
}

func (s *Store) Record(ctx context.Context, entry Entry) error {
	occurredAt := entry.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	if _, err := s.db.Exec(ctx, `
INSERT INTO gateway_audit_log (
    audit_id,
    occurred_at,
    caller_type,
    caller_id,
    caller_display,
    tool_name,
    source_service,
    sanitized_arguments_json,
    outcome,
    http_status,
    latency_ms,
    error_kind,
    result_summary_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13::jsonb)`,
		uuid.New(),
		occurredAt,
		entry.CallerType,
		entry.CallerID,
		entry.CallerDisplay,
		entry.ToolName,
		entry.SourceService,
		emptyJSON(entry.SanitizedArgumentsJSON),
		entry.Outcome,
		entry.HTTPStatus,
		entry.LatencyMS,
		entry.ErrorKind,
		emptyJSON(entry.ResultSummaryJSON),
	); err != nil {
		return fmt.Errorf("insert audit row: %w", err)
	}

	return nil
}

func emptyJSON(value string) string {
	if value == "" {
		return "{}"
	}

	var payload any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return "{}"
	}
	return value
}
