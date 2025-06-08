package main

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

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
		name    string
		fields  fields
		args    args
		wantErr bool
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
		})
	}
}
