package config

import "testing"

func TestTelegramConfigurationRequiresGroupWhenEnabled(t *testing.T) {
	cfg := TelegramConf{BotToken: "token", GroupChatIDs: ""}
	if _, err := cfg.AllowedGroupIDs(); err == nil {
		t.Fatal("expected enabled Telegram configuration without groups to fail")
	}
}

func TestTelegramConfigurationParsesGroupIDs(t *testing.T) {
	cfg := TelegramConf{BotToken: "token", GroupChatIDs: "-1001, -1002"}
	groups, err := cfg.AllowedGroupIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %#v", groups)
	}
	if _, ok := groups[-1002]; !ok {
		t.Fatalf("missing group -1002 in %#v", groups)
	}
}
