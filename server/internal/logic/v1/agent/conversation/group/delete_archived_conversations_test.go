package group

import (
	"testing"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
)

func TestArchivedThreadIDs(t *testing.T) {
	threads := []agentdomain.StoredThread{
		{ID: "active"},
		{ID: "archived", Archived: true},
	}

	got := archivedThreadIDs(threads)
	if _, ok := got["archived"]; !ok {
		t.Fatal("archived thread was not included")
	}
	if _, ok := got["active"]; ok {
		t.Fatal("active thread was included")
	}
}
