package agent

import (
	"encoding/json"
	"testing"
)

func TestAccountUpdatedNotificationCachesLoginStatus(t *testing.T) {
	runtime := &CodexAppServerRuntime{}
	runtime.handleNotification("account/updated", json.RawMessage(`{"authMode":"chatgpt","planType":"plus"}`), "1")

	status, ok := runtime.cachedAccountStatus()
	if !ok {
		t.Fatal("account status was not cached")
	}
	if !status.LoggedIn || status.AuthMode != "chatgpt" || status.PlanType != "plus" {
		t.Fatalf("unexpected account status: %+v", status)
	}
}

func TestAccountLoginCompletedCachesChatGPTStatus(t *testing.T) {
	runtime := &CodexAppServerRuntime{}
	runtime.handleNotification("account/login/completed", json.RawMessage(`{"loginId":"login-1","success":true}`), "1")

	status, ok := runtime.cachedAccountStatus()
	if !ok {
		t.Fatal("account status was not cached")
	}
	if !status.LoggedIn || status.AuthMode != "chatgpt" {
		t.Fatalf("unexpected account status: %+v", status)
	}
}

func TestAccountUpdatedNotificationClearsLoginStatus(t *testing.T) {
	runtime := &CodexAppServerRuntime{account: AccountStatus{AuthMode: "chatgpt", LoggedIn: true}, accountKnown: true}
	runtime.handleNotification("account/updated", json.RawMessage(`{"authMode":null}`), "1")

	status, ok := runtime.cachedAccountStatus()
	if !ok {
		t.Fatal("account status was not cached")
	}
	if status.LoggedIn || status.AuthMode != "" {
		t.Fatalf("unexpected account status: %+v", status)
	}
}
