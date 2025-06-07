package main

import (
	"context"

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
	clientset *kubernetes.Clientset,
) func(
	messages []anthropic.MessageParam,
	messageStore MessageStore,
) (string, error) {
	return func(
		messages []anthropic.MessageParam,
		messageStore MessageStore,
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
			switch contentPart.Type {
			case "text":
				content += contentPart.Text
				messageStore.AppendMessages(conversationID, []anthropic.MessageParam{
					anthropic.NewTextBlock(contentPart.Text),
				})
			case "tool_use":
				toolUse := contentPart.ToolUse
				if toolUse.Name == "get_pods" {
					pods, err := kubeClient.GetPods(toolUse.Input.Namespace)
				}
				messageStore.AppendMessages(conversationID, []anthropic.MessageParam{
					anthropic.NewToolUseMessage(
						anthropic.NewToolUseBlock(
							anthropic.NewToolUseBlockContent(toolUse.Name, toolUse.Input.Namespace),
						),
					),
				})
			}
		}
		return content, nil
	}

}

type MessageStore interface {
	CallLLM(conversationID string, text string) (string, error)
	AppendMessages(conversationID string, message []anthropic.MessageParam) error
}

type SlackMessageStore struct {
	messages map[string][]anthropic.MessageParam
	llm      func(messages []anthropic.MessageParam) (string, error)
}

func (s *SlackMessageStore) CallLLM(conversationID string, text string) (string, error) {
	s.messages[conversationID] = append(s.messages[conversationID], anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
	return s.llm(s.messages[conversationID])
}

func (s *SlackMessageStore) AppendMessages(conversationID string, message []anthropic.MessageParam) error {

func NewSlackMessageStore(llm func(messages []anthropic.MessageParam) (string, error)) *SlackMessageStore {
	return &SlackMessageStore{
		messages: make(map[string][]anthropic.MessageParam),
		llm:      llm,
	}
}
