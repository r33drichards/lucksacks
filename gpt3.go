package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

var (
	OpenAPISecretKey = os.Getenv("OPENAPI_SECRET_KEY")
)

type choice struct {
	text string
}

type gpt3Response struct {
	choices []choice
}

func gpt3(input string) (string, error) {
	url := "https://api.openai.com/v1/completions"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + OpenAPISecretKey,
	}
	body := []byte(`{"text": "` + input + `"}`)
	resp, err := postRequest(url, headers, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var gpt3Response gpt3Response
	err = json.NewDecoder(resp.Body).Decode(&gpt3Response)
	if err != nil {
		return "", err
	}
	return gpt3Response.choices[0].text, nil

}

// https://beta.openai.com/docs/api-reference/completions/create
// curl https://api.openai.com/v1/completions \
//   -H 'Content-Type: application/json' \
//   -H 'Authorization: Bearer YOUR_API_KEY' \
//   -d '{
//   "model": "text-davinci-002",
//   "prompt": "Say this is a test",
//   "max_tokens": 6,
//   "temperature": 0
// }'
// Response
//{
// 	"id": "cmpl-uqkvlQyYK7bGYrRHQ0eXlWi7",
// 	"object": "text_completion",
// 	"created": 1589478378,
// 	"model": "text-davinci-002",
// 	"choices": [
// 	  {
// 		"text": "\n\nThis is a test",
// 		"index": 0,
// 		"logprobs": null,
// 		"finish_reason": "length"
// 	  }
// 	],
// 	"usage": {
// 	  "prompt_tokens": 5,
// 	  "completion_tokens": 6,
// 	  "total_tokens": 11
// 	}
//   }

// post request to openai api with headers
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
