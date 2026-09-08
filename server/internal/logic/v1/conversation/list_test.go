package conversation

import (
	"testing"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
)

func TestFilterOwnedThreadsUsesListedThreadsAsSourceOfTruth(t *testing.T) {
	listed := []agentdomain.StoredThread{
		{ID: "active"},
		{ID: "archived", Archived: true},
		{ID: "other-user"},
	}
	owned := []*conversationmodel.Conversation{
		{Id: "active"},
		{Id: "archived"},
		{Id: "missing-from-thread-list"},
	}

	got := filterOwnedThreads(listed, owned)
	if len(got) != 2 {
		t.Fatalf("len(filterOwnedThreads()) = %d, want 2", len(got))
	}
	if got[0].ID != "active" || got[1].ID != "archived" {
		t.Fatalf("filterOwnedThreads() ids = [%s %s], want [active archived]", got[0].ID, got[1].ID)
	}
}
