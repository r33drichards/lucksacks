package main

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// Add this mock type to allow function-based mocking of LLMInterface
// mockLLM implements LLMInterface for testing

type mockLLM struct {
	fn func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (*LLMResponse, error)
}

func (m *mockLLM) Prompt(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (*LLMResponse, error) {
	return m.fn(messages, messageStore, conversationID)
}

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
		llm      LLMInterface
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
				llm: &mockLLM{fn: func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (*LLMResponse, error) {
					return &LLMResponse{Message: "test", Loop: false}, nil
				}},
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
				llm: &mockLLM{fn: func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (*LLMResponse, error) {
					return &LLMResponse{Message: "test", Loop: false}, nil
				}},
			},
			args: args{
				conversationID: "test",
				message: []anthropic.MessageParam{
					anthropic.NewAssistantMessage(anthropic.NewTextBlock("hello")),
				},
			},
			expectedMessages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
				anthropic.NewAssistantMessage(anthropic.NewTextBlock("hello")),
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
					if gotContent.OfText.Text != wantContent.OfText.Text {
						t.Errorf("SlackMessageStore.AppendMessages() = %v, want %v", gotContent, wantContent)
					}
					// if gotContent.OfThinking.Thinking != wantContent.OfThinking.Thinking {
					// 	t.Errorf("SlackMessageStore.AppendMessages() = %v, want %v", gotContent, wantContent)
					// }
					// if gotContent.OfToolResult.Content[0].OfText.Text != wantContent.OfToolResult.Content[0].OfText.Text {
					// 	t.Errorf("SlackMessageStore.AppendMessages() = %v, want %v", gotContent, wantContent)
					// }

				}
			}

		})
	}

}

// func Test_handleMessage(t *testing.T) {
// 	events := []string{
// 		`{"type": "message_start", "message": {}}`,
// 		`{"type": "content_block_start", "index": 0, "content_block": {"type": "tool_use", "id": "toolu_id", "name": "base64", "input": {}}}`,
// 		`{"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": "{\"text\":"}}`,
// 		`{"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": " \"test\"}"}}`,
// 		`{"type": "content_block_stop", "index": 0}`,
// 		`{"type": "message_stop"}`,
// 	}
// 	message := anthropic.Message{}
// 	for _, eventStr := range events {
// 		event := anthropic.MessageStreamEventUnion{}
// 		err := (&event).UnmarshalJSON([]byte(eventStr))
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		(&message).Accumulate(event)
// 	}

// 	type args struct {
// 		message        *anthropic.Message
// 		messageStore   MessageStore
// 		conversationID string
// 	}
// 	tests := []struct {
// 		name              string
// 		args              args
// 		want              *LLMResponse
// 		wantErr           bool
// 		messageStoreState map[string][]anthropic.MessageParam
// 	}{
// 		// TODO: Add test cases.
// 		{
// 			name: "test",
// 			args: args{
// 				message: &message,
// 				messageStore: &SlackMessageStore{
// 					messages: map[string][]anthropic.MessageParam{
// 						"test": {anthropic.NewUserMessage(anthropic.NewTextBlock("test"))},
// 					},
// 				},
// 				conversationID: "test",
// 			},
// 			want:    &LLMResponse{Message: "base64: \ndGVzdA==", Loop: true},
// 			wantErr: false,
// 			messageStoreState: map[string][]anthropic.MessageParam{
// 				"test": {
// 					anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
// 					anthropic.NewAssistantMessage(anthropic.NewToolUseBlock(
// 						"base64",
// 						map[string]interface{}{"text": "test"},
// 						"tool_use_id",
// 					)),
// 					anthropic.NewUserMessage(anthropic.NewToolResultBlock(
// 						"base64: {\"text\":\"test\"}\n\nbase64: \ndGVzdA==",
// 						"tool_use_id",
// 						true,
// 					)),
// 				},
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := handleMessage(
// 				tt.args.message,
// 				tt.args.messageStore,
// 				tt.args.conversationID,
// 			)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("handleMessage() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			diff := cmp.Diff(got, tt.want)
// 			if diff != "" {
// 				t.Errorf("handleMessage() = %v, want %v", got, tt.want)
// 			}
// 			// if !reflect.DeepEqual(tt.args.messageStore.GetMessages(), tt.messageStoreState) {
// 			// 	t.Errorf("handleMessage() messageStoreState = %v, want %v", tt.args.messageStore.GetMessages(), tt.messageStoreState)
// 			// }
// 		})
// 	}
// }
