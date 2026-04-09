package audit

import (
	"context"
	"testing"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/testutil"
)

func TestStoreRecordsAuditRowsInPostgres(t *testing.T) {
	ctx := context.Background()
	postgresEnv, err := testutil.StartPostgres(ctx)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := postgresEnv.Close(); closeErr != nil {
			t.Fatalf("PostgresEnv.Close() error = %v", closeErr)
		}
	})

	store, err := Open(ctx, postgresEnv.DatabaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(store.Close)

	err = store.Record(ctx, Entry{
		OccurredAt:             time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC),
		CallerType:             "automation",
		CallerID:               "ci-bot",
		CallerDisplay:          "CI Bot",
		ToolName:               "athena.get_current_zone_occupancy",
		SourceService:          "athena",
		SanitizedArgumentsJSON: `{"facility_id":"ashtonbee","zone_id":"gym-floor"}`,
		Outcome:                "success",
		HTTPStatus:             200,
		LatencyMS:              17,
		ErrorKind:              "",
		ResultSummaryJSON:      `{"facility_id":"ashtonbee","zone_id":"gym-floor","current_count":4}`,
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	var count int
	if err := postgresEnv.DB.QueryRow(ctx, `SELECT count(*) FROM gateway_audit_log`).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("audit row count = %d, want 1", count)
	}

	var callerType string
	var toolName string
	if err := postgresEnv.DB.QueryRow(ctx, `SELECT caller_type, tool_name FROM gateway_audit_log LIMIT 1`).Scan(&callerType, &toolName); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if callerType != "automation" {
		t.Fatalf("caller_type = %q, want automation", callerType)
	}
	if toolName != "athena.get_current_zone_occupancy" {
		t.Fatalf("tool_name = %q, want athena.get_current_zone_occupancy", toolName)
	}
}
