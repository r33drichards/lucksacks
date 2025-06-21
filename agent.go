package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"

	"github.com/anthropics/anthropic-sdk-go"
)

type ToolHandler interface {
	GetName() string
	HandleTool(
		input json.RawMessage,
	) (*string, error)
}

func newTemplateToolHandler(
	name string,
	handleTool func(input json.RawMessage) (*string, error),
) ToolHandler {
	return &templateToolHandler{
		name:       name,
		handleTool: handleTool,
	}
}

type templateToolHandler struct {
	name       string
	handleTool func(input json.RawMessage) (*string, error)
}

func (h *templateToolHandler) GetName() string {
	return h.name
}

func (h *templateToolHandler) HandleTool(input json.RawMessage) (*string, error) {
	return h.handleTool(input)
}

func CreateToolHandler[T any](
	name string,
	handleTool func(input T) (*string, error),
) ToolHandler {
	handler := func(input json.RawMessage) (*string, error) {
		var toolInput T
		err := json.Unmarshal(input, &toolInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal input")
		}
		return handleTool(toolInput)
	}
	return newTemplateToolHandler(name, handler)
}

type messageHandler interface {
	HandleMessage(
		message *anthropic.Message,
		messageStore MessageStore,
		conversationID string,
	) (*LLMResponse, error)
}

type AnthropicMessageHandler struct {
	tools map[string]ToolHandler
}

func NewAnthropicMessageHandler(
	tools []ToolHandler,
) *AnthropicMessageHandler {
	toolsMap := make(map[string]ToolHandler)
	for _, tool := range tools {
		toolsMap[tool.GetName()] = tool
	}
	return &AnthropicMessageHandler{
		tools: toolsMap,
	}
}

func (h *AnthropicMessageHandler) getTool(
	name string,
) ToolHandler {
	if tool, ok := h.tools[name]; ok {
		return tool
	}
	return nil
}

func (h *AnthropicMessageHandler) callTool(
	name string,
	input json.RawMessage,
) (*string, error) {
	tool := h.getTool(name)
	if tool == nil {
		return nil, errors.New("tool not found")
	}
	return tool.HandleTool(input)
}

func (h *AnthropicMessageHandler) HandleMessage(
	message *anthropic.Message,
	messageStore MessageStore,
	conversationID string,
) (*LLMResponse, error) {

	content := ""

	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			content += block.Text
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

			maybeResponse, err := h.callTool(block.Name, variant.Input)
			if err != nil {
				return nil, errors.Wrap(err, "failed to call tool")
			}

			if maybeResponse == nil {
				return nil, errors.New("tool returned nil")
			}

			response := *maybeResponse

			content += "\n" + block.Name + ": \n" + response
			content = strings.TrimSpace(content)
			response = strings.TrimSpace(response)
			toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, response, false))

		}
	}

	mesagesToStore := []anthropic.MessageParam{message.ToParam()}
	if len(toolResults) > 0 {
		mesagesToStore = append(mesagesToStore, anthropic.NewUserMessage(toolResults...))
	}
	if strings.TrimSpace(content) != "" {
		mesagesToStore = append(mesagesToStore, anthropic.NewAssistantMessage(anthropic.NewTextBlock(strings.TrimSpace(content))))
	}

	messageStore.AppendMessages(conversationID, mesagesToStore)
	return &LLMResponse{
		Message: content,
		Loop:    len(toolResults) > 0,
	}, nil

}

// LLMInterface defines the interface for LLMs with a Prompt method.
type LLMInterface interface {
	Prompt(
		messages []anthropic.MessageParam,
		messageStore MessageStore,
		conversationID string,
	) (*LLMResponse, error)
}

// LLM is a struct that holds the anthropic client and any other config.
type LLM struct {
	client         anthropic.Client
	messageHandler messageHandler
}

func newLLM(
	client anthropic.Client,
	messageHandler messageHandler,
) *LLM {
	return &LLM{
		client:         client,
		messageHandler: messageHandler,
	}
}

// Prompt implements the LLMInterface for LLM.
func (l *LLM) Prompt(messages []anthropic.MessageParam, messageStore MessageStore, conversationID string) (*LLMResponse, error) {
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
						"type": "string",
						"description": `The JavaScript code to run. console.log does not work. return the value at the end of the script to get the outcome of the sciprt
console.log("hello world") // does not work 
"hello" // this works


the above script would return "hello"
						`,
					},
				},
			},
		},
	}

	tools := make([]anthropic.ToolUnionParam, len(toolParams))
	for i, toolParam := range toolParams {
		tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	message, err := l.client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude4Sonnet20250514,
		MaxTokens: 20_000,
		Messages:  messages,
		Tools:     tools,
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfEnabled: &anthropic.ThinkingConfigEnabledParam{BudgetTokens: 5_000}},
		System: []anthropic.TextBlockParam{
			{
				Text: "your responses are going to be going to slack, so use that format for your responses",
			},
			{
				Text: "text like this **bold** is not supported in slack, it just shows the starts, so use a different way to organize your text",
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "couldn't create message")
	}
	resp, err := l.messageHandler.HandleMessage(message, messageStore, conversationID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't handle message")
	}
	return resp, nil
}

// NewLLM returns a new LLM struct implementing LLMInterface.
func NewLLM(client anthropic.Client, messageHandler messageHandler) *LLM {
	return &LLM{client: client, messageHandler: messageHandler}
}

type LLMResponse struct {
	Message string
	Loop    bool
}

type MessageStore interface {
	CallLLM(conversationID string, text string) (*LLMResponse, error)
	AppendMessages(conversationID string, message []anthropic.MessageParam) error
	GetMessages() map[string][]anthropic.MessageParam
	Loop(
		conversationID string,
		api *slack.Client,
		reqID string,
	) (*LLMResponse, error)
}

var _ MessageStore = &SlackMessageStore{}

type SlackMessageStore struct {
	messages map[string][]anthropic.MessageParam
	llm      LLMInterface
}

func (s *SlackMessageStore) CallLLM(conversationID string, text string) (*LLMResponse, error) {
	if strings.TrimSpace(text) != "" {
		s.messages[conversationID] = append(s.messages[conversationID], anthropic.NewUserMessage(anthropic.NewTextBlock(strings.TrimSpace(text))))
	}

	if len(s.messages[conversationID]) == 0 {
		return &LLMResponse{
			Message: "I can't respond to an empty message. Please provide some input. keep your outputs basic and text only since no formatting is applied.",
			Loop:    false,
		}, nil
	}
	message, err := s.llm.Prompt(
		s.messages[conversationID],
		s,
		conversationID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't call LLM")
	}

	return message, nil
}

func (s *SlackMessageStore) AppendMessages(conversationID string, message []anthropic.MessageParam) error {
	s.messages[conversationID] = append(s.messages[conversationID], message...)
	return nil
}

func (s *SlackMessageStore) GetMessages() map[string][]anthropic.MessageParam {
	return s.messages
}

func (s *SlackMessageStore) Loop(
	conversationID string,
	api *slack.Client,
	reqID string,
) (*LLMResponse, error) {
	message, err := s.llm.Prompt(
		s.messages[conversationID],
		s,
		conversationID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't call LLM")
	}

	return message, nil
}

func NewSlackMessageStore(
	llm LLMInterface,
) *SlackMessageStore {
	return &SlackMessageStore{
		messages: make(map[string][]anthropic.MessageParam),
		llm:      llm,
	}
}
