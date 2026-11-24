package qa

import (
	"context"
	"strings"
	"testing"
	"time"

	conversation "github.com/jzero-io/agentrazor/core-engine/sdk/agent"

	"telegram_chat_bot/internal/telegrambot"
)

type fakeConversationClient struct {
	requests []conversation.Request
}

func (f *fakeConversationClient) Chat(_ context.Context, request conversation.Request) (*conversation.Response, error) {
	f.requests = append(f.requests, request)
	return &conversation.Response{ConversationID: request.ConversationID, Answer: "answer"}, nil
}

func (*fakeConversationClient) CreateConversation(context.Context, conversation.CreateConversationRequest) (*conversation.Conversation, error) {
	return &conversation.Conversation{ID: "conversation-1"}, nil
}

func (*fakeConversationClient) ListGroups(context.Context) ([]conversation.Group, error) {
	return nil, nil
}

func (*fakeConversationClient) CreateGroup(context.Context, string) (*conversation.Group, error) {
	return &conversation.Group{ID: "group-1", Name: GroupName}, nil
}

func (*fakeConversationClient) UpdateConversation(context.Context, string, conversation.UpdateConversationRequest) (*conversation.Conversation, error) {
	return &conversation.Conversation{ID: "conversation-1"}, nil
}

type fakeSender struct {
	text string
}

func (f *fakeSender) SendMessage(_ context.Context, _ int64, _ int64, text string) error {
	f.text = text
	return nil
}

type fakeStore struct {
	conversationID string
	saved          bool
	marked         int64
}

func (f *fakeStore) FindConversation(context.Context, int64) (string, error) {
	return f.conversationID, nil
}

func (f *fakeStore) SaveConversation(_ context.Context, _ int64, conversationID, _ string) error {
	f.saved = true
	f.conversationID = conversationID
	return nil
}

func (f *fakeStore) MarkUpdateProcessed(_ context.Context, updateID int64) (bool, error) {
	f.marked = updateID
	return true, nil
}

func (*fakeStore) PruneProcessedUpdates(context.Context, time.Time) error {
	return nil
}

func TestHandleCreatesConversationAndSendsAnswer(t *testing.T) {
	client := &fakeConversationClient{}
	sender := &fakeSender{}
	store := &fakeStore{}
	service, err := New(client, sender, store, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = service.Handle(context.Background(), telegrambot.Question{
		UpdateID: 42, ChatID: -1001, MessageID: 99, UserID: 7, ChatTitle: "Support", Username: "customer", Text: "pricing?",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !store.saved || store.conversationID != "conversation-1" || store.marked != 42 {
		t.Fatalf("unexpected store state: %#v", store)
	}
	if sender.text != "answer" {
		t.Fatalf("unexpected answer %q", sender.text)
	}
	if len(client.requests) != 1 || !strings.Contains(client.requests[0].Content, "$"+SkillName) {
		t.Fatalf("unexpected AgentRazor request: %#v", client.requests)
	}
}
