package telegrambot

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestStripBotMention(t *testing.T) {
	text := "  你好 @BlockRazorBot  请问 BSC Private Mempool 的价格？"
	username := "@blockrazorbot"
	offset := len(utf16.Encode([]rune("  你好 ")))
	length := len(utf16.Encode([]rune("@BlockRazorBot")))
	got, ok := stripBotMention(text, []Entity{{Type: "mention", Offset: offset, Length: length}}, username)
	if !ok || got != "你好   请问 BSC Private Mempool 的价格？" {
		t.Fatalf("stripBotMention = %q, %v", got, ok)
	}
}

func TestStripBotMentionRequiresMatchingBot(t *testing.T) {
	text := "hello @SomeoneElse"
	_, ok := stripBotMention(text, []Entity{{
		Type: "mention", Offset: len(utf16.Encode([]rune("hello "))), Length: len(utf16.Encode([]rune("@SomeoneElse"))),
	}}, "blockrazorbot")
	if ok {
		t.Fatal("expected a mention for another bot to be ignored")
	}
}

func TestSplitMessageLimitsUTF16Units(t *testing.T) {
	content := strings.Repeat("🙂", 3000)
	chunks := SplitMessage(content, maxMessageUnits)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if got := strings.Join(chunks, ""); got != content {
		t.Fatal("split content does not match the original")
	}
	for index, chunk := range chunks {
		if units := len(utf16.Encode([]rune(chunk))); units > maxMessageUnits {
			t.Fatalf("chunk %d has %d UTF-16 units", index, units)
		}
	}
}
