package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rosbit/go-quickjs"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

func handleMessage(
	message *anthropic.Message,
	messageStore MessageStore,
	conversationID string,
) (string, error) {
	content := ""

	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			content += block.Text
			content += "\n"

		case anthropic.ToolUseBlock:
			inputJSON, _ := json.Marshal(block.Input)
			content += block.Name + ": " + string(inputJSON)
			content += "\n"
		case anthropic.ThinkingBlock:
			content += block.Thinking
			content += "\n"
		}
	}

	toolResults := []anthropic.ContentBlockParamUnion{}
	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:

			var response string
			switch block.Name {
			case "base64":
				var input struct {
					Text string `json:"text"`
				}

				raw := variant.Input

				err := json.Unmarshal(raw, &input)

				if err != nil {
					return "", errors.Wrap(err, "failed to unmarshal input")
				}

				response = base64.StdEncoding.EncodeToString([]byte(input.Text))
			case "jwtdecode":
				var input struct {
					Token string `json:"token"`
				}

				err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input)
				if err != nil {
					return "", errors.Wrap(err, "failed to unmarshal input")
				}

				response, err = jwtdecode(input.Token)
				if err != nil {
					return "", errors.Wrap(err, "failed to decode JWT")
				}
			case "uuid":
				response = uuid.New().String()
			case "quickjs":
				var input struct {
					Code string `json:"code"`
				}

				err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input)
				if err != nil {
					return "", errors.Wrap(err, "failed to create context")
				}
				ctx, err := quickjs.NewContext()
				if err != nil {
					return "", errors.Wrap(err, "failed to create context")
				}

				res, err := ctx.Eval(input.Code, nil)
				if err != nil {
					return "", errors.Wrap(err, "failed to evaluate code")
				}
				response = fmt.Sprintf("%v", res)
			default:
				response = "Unknown tool: " + block.Name
			}

			content += "\n" + block.Name + ": \n" + response
			toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, response, false))

		}
	}

	if len(toolResults) == 0 {
		return content, nil
	}

	mesagesToStore := []anthropic.MessageParam{message.ToParam()}
	mesagesToStore = append(mesagesToStore, anthropic.NewUserMessage(toolResults...))
	messageStore.AppendMessages(conversationID, mesagesToStore)
	messagesToStore := []anthropic.MessageParam{message.ToParam()}
	messagesToStore = append(messagesToStore, anthropic.NewAssistantMessage(anthropic.NewTextBlock(content)))
	messageStore.AppendMessages(conversationID, messagesToStore)
	return content, nil
}

func NewLLM(
	client anthropic.Client,
) func(
	messages []anthropic.MessageParam,
	messageStore MessageStore,
	conversationID string,
) (string, error) {
	return func(
		messages []anthropic.MessageParam,
		messageStore MessageStore,
		conversationID string,
	) (string, error) {

		toolParams := []anthropic.ToolParam{
			{
				Name:        "base64",
				Description: anthropic.String("Base64 encode a string"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"text": map[string]interface{}{
							"type":        "string",
							"description": "The text to encode",
						},
					},
				},
			},
			{
				Name:        "jwtdecode",
				Description: anthropic.String("Decode a JWT token"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"token": map[string]interface{}{
							"type":        "string",
							"description": "The JWT token to decode",
						},
					},
				},
			},
			{
				Name:        "uuid",
				Description: anthropic.String("Generate a UUID"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{},
				},
			},
			{
				Name:        "quickjs",
				Description: anthropic.String("Run a JavaScript function"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "The JavaScript code to run",
						},
					},
				},
			},
		}

		tools := make([]anthropic.ToolUnionParam, len(toolParams))
		for i, toolParam := range toolParams {
			tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
		}

		message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaude4Sonnet20250514,
			MaxTokens: 20_000,
			Messages:  messages,
			Tools:     tools,
			Thinking: anthropic.ThinkingConfigParamUnion{
				OfEnabled: &anthropic.ThinkingConfigEnabledParam{BudgetTokens: 1024}},
		})

		if err != nil {
			return "", err
		}
		content, err := handleMessage(message, messageStore, conversationID)
		if err != nil {
			return "", errors.Wrap(err, "couldn't handle message")
		}
		return content, nil
	}

}

type MessageStore interface {
	CallLLM(conversationID string, text string) (string, error)
	AppendMessages(conversationID string, message []anthropic.MessageParam) error
}

var _ MessageStore = &SlackMessageStore{}

type SlackMessageStore struct {
	messages map[string][]anthropic.MessageParam
	llm      func(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (string, error)
}

func (s *SlackMessageStore) CallLLM(conversationID string, text string) (string, error) {
	s.messages[conversationID] = append(s.messages[conversationID], anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
	return s.llm(s.messages[conversationID], s, conversationID)
}

func (s *SlackMessageStore) AppendMessages(conversationID string, message []anthropic.MessageParam) error {
	s.messages[conversationID] = append(s.messages[conversationID], message...)
	return nil
}

func NewSlackMessageStore(
	llm func(
		messages []anthropic.MessageParam,
		messageStore MessageStore,
		conversationID string,
	) (string, error),
) *SlackMessageStore {
	return &SlackMessageStore{
		messages: make(map[string][]anthropic.MessageParam),
		llm:      llm,
	}
}
