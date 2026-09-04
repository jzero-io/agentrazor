package conversation

import (
	"testing"
	"time"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
)

func TestBuildConversationTrendByDay(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.September, 4, 16, 0, 0, 0, location)
	threads := []agentdomain.StoredThread{
		{ID: "active", CreatedAt: time.Date(2026, time.September, 3, 10, 0, 0, 0, location)},
		{
			ID:        "archived",
			Archived:  true,
			CreatedAt: time.Date(2026, time.September, 2, 10, 0, 0, 0, location),
			UpdatedAt: time.Date(2026, time.September, 4, 10, 0, 0, 0, location),
		},
	}

	points := buildConversationTrend(threads, "day", now)
	if len(points) != 30 {
		t.Fatalf("got %d points, want 30", len(points))
	}
	if points[27].Period != "2026-09-02" || points[27].TotalConversations != 1 {
		t.Fatalf("unexpected September 2 point: %#v", points[27])
	}
	if points[28].Period != "2026-09-03" || points[28].TotalConversations != 1 {
		t.Fatalf("unexpected September 3 point: %#v", points[28])
	}
	if points[29].Period != "2026-09-04" || points[29].ArchivedConversations != 1 {
		t.Fatalf("unexpected September 4 point: %#v", points[29])
	}
}

func TestBuildConversationTrendByMonth(t *testing.T) {
	now := time.Date(2026, time.September, 4, 16, 0, 0, 0, time.UTC)
	threads := []agentdomain.StoredThread{{
		ID:        "archived",
		Archived:  true,
		CreatedAt: time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
	}}

	points := buildConversationTrend(threads, "month", now)
	if len(points) != 12 {
		t.Fatalf("got %d points, want 12", len(points))
	}
	if points[10].Period != "2026-08" || points[10].TotalConversations != 1 {
		t.Fatalf("unexpected August point: %#v", points[10])
	}
	if points[11].Period != "2026-09" || points[11].ArchivedConversations != 1 {
		t.Fatalf("unexpected September point: %#v", points[11])
	}
}
