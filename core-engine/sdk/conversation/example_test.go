package conversation_test

import (
	"context"
	"fmt"
	"log"

	"github.com/jzero-io/agentrazor/core-engine/sdk/conversation"
)

func ExampleClient_Chat() {
	client, err := conversation.NewClient(conversation.Config{
		BaseURL: "https://agent.example.com",
		APIKey:  "ar-your-api-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Chat(context.Background(), conversation.Request{
		Content: "你好，请介绍一下你自己。",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Answer)

	// Continue the same conversation by passing response.ConversationID.
}
