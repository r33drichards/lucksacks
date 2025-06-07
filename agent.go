package main

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubernetesClient interface {
	GetPods(namespace string) ([]corev1.Pod, error)
	Namespaces() ([]corev1.Namespace, error)
}

type KubernetesClientImpl struct {
	clientset *kubernetes.Clientset
}

func (c *KubernetesClientImpl) GetPods(namespace string) ([]corev1.Pod, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (c *KubernetesClientImpl) Namespaces() ([]corev1.Namespace, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return namespaces.Items, nil
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
			Thinking:  anthropic.ThinkingConfigParamUnion{OfEnabled: &anthropic.ThinkingConfigEnabledParam{BudgetTokens: 1024}},
		})

		if err != nil {
			return "", err
		}
		// collect all the content from the message
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

		messagesToStore := []anthropic.MessageParam{message.ToParam()}
		messagesToStore = append(messagesToStore, anthropic.NewAssistantMessage(anthropic.NewTextBlock(content)))
		messageStore.AppendMessages(conversationID, messagesToStore)

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range message.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.ToolUseBlock:

				var response interface{}
				switch block.Name {
				case "base64":
					var input struct {
						Text string `json:"text"`
					}

					err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input)
					if err != nil {
						panic(err)
					}

					response = base64.StdEncoding.EncodeToString([]byte(input.Text))
				}

				b, err := json.Marshal(response)
				if err != nil {
					panic(err)
				}

				content += "\n" + block.Name + ": \n" + string(b)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, string(b), false))

			}
		}

		if len(toolResults) == 0 {
			return content, nil
		}

		messagesToStore = append(messagesToStore, anthropic.NewAssistantMessage(toolResults...))
		messageStore.AppendMessages(conversationID, messagesToStore)

		return content, nil
	}

}

type MessageStore interface {
	CallLLM(conversationID string, text string) (string, error)
	AppendMessages(conversationID string, message []anthropic.MessageParam) error
}

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
