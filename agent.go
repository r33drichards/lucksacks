package main

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

func NewLLM(
	client anthropic.Client,
) func(
	messages []anthropic.MessageParam,
) (string, error) {
	return func(
		messages []anthropic.MessageParam,
	) (string, error) {
		toolParams := []anthropic.ToolParam{}

		tools := make([]anthropic.ToolUnionParam, len(toolParams))
		for i, toolParam := range toolParams {
			tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
		}

		message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaude4Sonnet20250514,
			MaxTokens: 20_000,
			Messages:  messages,
			Tools:     tools,
			Thinking:  anthropic.ThinkingConfigParamUnion{OfEnabled: &anthropic.ThinkingConfigEnabledParam{BudgetTokens: 1024}},
		})

		if err != nil {
			return "", err
		}
		// collect all the content from the message
		content := ""
		for _, contentPart := range message.Content {
			content += contentPart.Text
		}
		return content, nil
	}

}

type MessageStore interface {
	CallLLM(conversationID string, text string) (string, error)
}

type SlackMessageStore struct {
	messages map[string][]anthropic.MessageParam
	llm      func(messages []anthropic.MessageParam) (string, error)
}

func (s *SlackMessageStore) CallLLM(conversationID string, text string) (string, error) {
	s.messages[conversationID] = append(s.messages[conversationID], anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
	return s.llm(s.messages[conversationID])
}

func NewSlackMessageStore(llm func(messages []anthropic.MessageParam) (string, error)) *SlackMessageStore {
	return &SlackMessageStore{
		messages: make(map[string][]anthropic.MessageParam),
		llm:      llm,
	}
}
