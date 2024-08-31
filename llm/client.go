package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	client  *http.Client
	token   string
	baseURL string
	logger  Logger
}

func NewClient(token string) *Client {
	return &Client{
		client:  &http.Client{},
		token:   token,
		baseURL: "https://openrouter.ai/api/v1/",
	}
}

func mergeResponse(base, update *Response) *Response {
	if base == nil {
		base = &Response{}
		*base = *update

		base.Object, _ = strings.CutSuffix(base.Object, ".chunk")

		base.Choices = make([]Choice, len(update.Choices))

		for i := range base.Choices {
			base.Choices[i] = update.Choices[i]
			base.Choices[i].Message = &Message{}
			*base.Choices[i].Message = *base.Choices[i].Delta
			base.Choices[i].Delta = nil
		}
		return base
	}

	// fmt.Println(base.Choices, update.Choices)

	for i, choice := range update.Choices {
		base.Choices[i].Message.Content += choice.Delta.Content
		if base.Choices[i].FinishReason == "" {
			base.Choices[i].FinishReason = choice.FinishReason
		}
	}

	if base.Usage == nil && update.Usage != nil {
		base.Usage = &Usage{}
		*base.Usage = *update.Usage
	}

	return base
}

func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

func (c *Client) SendStreamRequest(req *Request, chunkChan chan<- *Response) (*Response, error) {
	req.Stream = true
	reqURL := c.baseURL + "chat/completions"

	httpResp, err := c.sendRequest(req, reqURL)
	if err != nil {
		return nil, err
	}

	contentType := httpResp.Header.Get("Content-Type")

	if contentType != "text/event-stream" {
		return nil, fmt.Errorf("expected stream, got %s", contentType)
	}

	errChan := make(chan error, 1)
	responseChan := make(chan *Response, 1)

	go func() {
		defer close(errChan)
		defer close(responseChan)

		response, err := func() (*Response, error) {
			defer close(chunkChan)

			var response *Response

			reader := NewSSEReader(httpResp.Body)

			for {
				event, err := reader.ReadEvent()
				if err != nil {
					if err != io.EOF {
						errChan <- err
					}

					return response, nil
				}

				if event.Event != "" {
					continue
				}

				data := event.Data

				if c.logger != nil {
					c.logger.Log("Chunk: ", data)
				}

				if data == "[DONE]" {
					return response, nil
				}

				var resp Response
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					return nil, err
				}

				response = mergeResponse(response, &resp)

				chunkChan <- &resp
			}
		}()
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	}()

	select {
	case err := <-errChan:
		return nil, err
	case resp := <-responseChan:
		return resp, nil
	}
}

func (c *Client) SendRequest(req *Request) (*Response, error) {
	req.Stream = false
	reqURL := c.baseURL + "chat/completions"

	httpResp, err := c.sendRequest(req, reqURL)
	if err != nil {
		return nil, err
	}

	defer httpResp.Body.Close()

	contentType := httpResp.Header.Get("Content-Type")

	if contentType != "application/json" {
		return nil, fmt.Errorf("expected application/json, got %s", contentType)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if c.logger != nil {
		c.logger.Log("Response: ", string(body))
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) sendRequest(req *Request, reqURL string) (*http.Response, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if c.logger != nil {
		c.logger.Log("Request: ", string(reqJSON))
	}

	httpReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqJSON))

	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	accept := []string{"application/json"}
	if req.Stream {
		accept = append(accept, "text/event-stream")
	}
	httpReq.Header.Set("Accept", strings.Join(accept, ", "))

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}
