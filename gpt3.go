package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type choice struct {
	Text string `json:"text"`
}

type gpt3Response struct {
	Choices []choice `json:"choices"`
}

type gpt3Request struct {
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

func newGpt3Request(prompt string) *gpt3Request {
	return &gpt3Request{
		Prompt:    prompt,
		Model:     "text-davinci-002",
		MaxTokens: 2048,
	}
}

func gpt3(input string) (string, error) {
	url := "https://api.openai.com/v1/completions"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + OpenAPISecretKey,
	}
	request := newGpt3Request(input)
	b, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	resp, err := postRequest(url, headers, b)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var gpt3Response gpt3Response
	err = json.NewDecoder(resp.Body).Decode(&gpt3Response)
	if err != nil {
		return "", err
	}
	fullresponse := ""
	for _, choice := range gpt3Response.Choices {
		fullresponse += choice.Text
	}
	return fullresponse, nil

}

func postRequest(url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	return client.Do(req)
}
