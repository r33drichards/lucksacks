package main

import (
	"fmt"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func zip[T any](a, b []T) [][]T {
	acc := make([][]T, 0, len(a))
	for i := 0; i < len(a); i++ {
		acc = append(acc, []T{a[i], b[i]})
	}
	return acc
}

func TestSlackMessageStore_AppendMessages(t *testing.T) {
	type fields struct {
		messages map[string][]anthropic.MessageParam
		llm      func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (string, error)
	}
	type args struct {
		conversationID string
		message        []anthropic.MessageParam
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantErr          bool
		expectedMessages []anthropic.MessageParam
	}{
		// TODO: Add test cases.
		{
			name: "test",
			fields: fields{
				messages: make(map[string][]anthropic.MessageParam),
				llm: func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (string, error) {
					return "test", nil
				},
			},
			args: args{
				conversationID: "test",
				message:        []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("test"))},
			},
			wantErr: false,
			expectedMessages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
			},
		},
		// append messages
		{
			name: "append messages",
			fields: fields{
				messages: map[string][]anthropic.MessageParam{
					"test": {anthropic.NewUserMessage(anthropic.NewTextBlock("test"))},
				},
				llm: func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (string, error) {
					return "test", nil
				},
			},
			args: args{
				conversationID: "test",
				message: []anthropic.MessageParam{
					anthropic.NewAssistantMessage(anthropic.NewTextBlock("test")),
				},
			},
			expectedMessages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
				anthropic.NewAssistantMessage(anthropic.NewTextBlock("test")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SlackMessageStore{
				messages: tt.fields.messages,
				llm:      tt.fields.llm,
			}
			if err := s.AppendMessages(tt.args.conversationID, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("SlackMessageStore.AppendMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
			gotMessages := zip(tt.fields.messages[tt.args.conversationID], tt.expectedMessages)
			for _, messages := range gotMessages {
				got := messages[0]
				want := messages[1]
				if got.Role != want.Role {
					t.Errorf("SlackMessageStore.AppendMessages() = %v, want %v", got, want)
				}
				zippedContent := zip(got.Content, want.Content)
				for _, content := range zippedContent {
					gotContent := content[0]
					wantContent := content[1]
					fmt.Printf("gotContent: %v\n", gotContent)
					fmt.Printf("wantContent: %v\n", wantContent)
				}
			}

		})
	}
}
