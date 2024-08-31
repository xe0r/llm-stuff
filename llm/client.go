package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type Client struct {
	client  *http.Client
	token   string
	baseURL string
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

		defer close(chunkChan)

		var response *Response

		reader := bufio.NewReader(httpResp.Body)

		kind := ""
		data := ""
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}

				responseChan <- response
				return
			}

			lineStr := strings.TrimSpace(string(line))

			if len(lineStr) != 0 {
				lineKind, content, _ := strings.Cut(lineStr, ": ")
				if kind == "" {
					kind = lineKind
				} else if kind != lineKind {
					errChan <- fmt.Errorf("expected %s, got %s", kind, lineKind)
					return
				}

				data += content
				continue
			}

			logMessage("Chunk: ", kind, data)

			if kind != "data" {
				//fmt.Printf("%s: %s\n", kind, data)
				kind, data = "", ""
				continue
			}

			if data == "[DONE]" {
				responseChan <- response
				return
			}

			//fmt.Printf("%s\n", data)

			var resp Response
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				errChan <- err
				return
			}

			kind, data = "", ""

			response = mergeResponse(response, &resp)

			chunkChan <- &resp
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

	logMessage("Response: ", string(body))

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

	logMessage("Request: ", string(reqJSON))

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

type ChatClient[T any] struct {
	client   *Client
	funcs    []CallableFunction
	funcsMap map[string]CallableFunction
	req      *Request
}

func NewChatClient(token string, funcs []CallableFunction) *ChatClient[string] {
	return NewChatClientWithType[string](token, funcs)
}

func NewChatClientWithType[T any](token string, funcs []CallableFunction) *ChatClient[T] {
	client := NewClient(token)

	tools := make([]Tool, 0, len(funcs))
	funcsMap := make(map[string]CallableFunction)

	for _, fn := range funcs {
		funcsMap[fn.GetName()] = fn
		tools = append(tools, Tool{
			Type: "function",
			Function: FunctionDescription{
				Name:        fn.GetName(),
				Description: fn.GetDescription(),
				Parameters:  fn.GetParameters(),
			},
		})
	}

	toolChoice := ""
	if len(tools) > 0 {
		toolChoice = "auto"
	}

	req := &Request{
		Tools:      tools,
		ToolChoice: toolChoice,
	}

	ty := reflect.TypeOf((*T)(nil)).Elem()

	if ty.Kind() == reflect.String {
		req.ResponseFormat.Type = "text"
	} else if ty.Kind() == reflect.Map {
		req.ResponseFormat.Type = "json_object"
	} else {
		req.ResponseFormat.Type = "json_schema"
		req.ResponseFormat.JSONSchema = &JSONSchema{
			Name:   "response",
			Strict: true,
			Schema: getParamDef(ty),
		}
	}

	return &ChatClient[T]{
		client: client,

		funcs:    funcs,
		funcsMap: funcsMap,
		req:      req,
	}
}

func (c *ChatClient[T]) SetModel(model string) {
	c.req.Model = model
}

// Some models don't support JSON Schema, or generate it incorrectly
func (c *ChatClient[T]) SetObjectResponse() {
	c.req.ResponseFormat.Type = "json_object"
	c.req.ResponseFormat.JSONSchema = nil
}

func (c *ChatClient[T]) AddMessage(role string, content string) {
	c.req.Messages = append(c.req.Messages, Message{
		Role:    role,
		Content: content,
	})
}

func (c *ChatClient[T]) GetResponse(chunkChan chan<- string) (T, error) {
	if chunkChan != nil {
		defer close(chunkChan)
	}

	result := *new(T)
	if c.req.Model == "" {
		return result, fmt.Errorf("model not set")
	}
	for {
		var resp *Response
		var err error
		if chunkChan != nil {
			subChunkChan := make(chan *Response)

			go func() {
				for chunk := range subChunkChan {
					if len(chunk.Choices) > 0 {
						chunkChan <- chunk.Choices[0].Delta.Content
					}
				}
			}()
			resp, err = c.client.SendStreamRequest(c.req, subChunkChan)
		} else {
			resp, err = c.client.SendRequest(c.req)
		}
		if err != nil {
			return result, err
		}

		if resp.Code != 0 {
			return result, fmt.Errorf("error code %d", resp.Code)
		}

		if resp.Error.Message != "" {
			return result, fmt.Errorf("error: %s", resp.Error.Message)
		}

		if len(resp.Choices) == 0 {
			return result, fmt.Errorf("no choices")
		}

		if len(resp.Choices) != 1 {
			fmt.Printf("multiple choices: %#v\n", resp.Choices)
		}

		choice := resp.Choices[0]

		if choice.Message == nil {
			return result, fmt.Errorf("no message")
		}

		c.req.Messages = append(c.req.Messages, *choice.Message)

		finishReason := strings.ToLower(choice.FinishReason)

		if finishReason == "stop" {
			tv := reflect.ValueOf(&result).Elem()
			if tv.Kind() == reflect.String {
				tv.SetString(choice.Message.Content)
				return result, nil
			}

			if err := json.Unmarshal([]byte(choice.Message.Content), &result); err != nil {
				return result, err
			}
			return result, nil
		}

		if finishReason == "tool_calls" {
			err := c.handleToolCalls(choice.Message.ToolCalls)
			if err != nil {
				return result, err
			}

			continue
		}

		return result, fmt.Errorf("unknown finish reason %s", finishReason)

	}
}

func (c *ChatClient[T]) handleToolCalls(toolCalls []ToolCall) error {
	for _, toolCall := range toolCalls {
		fn, ok := c.funcsMap[toolCall.Function.Name]
		if !ok {
			return fmt.Errorf("unknown function %s", toolCall.Function.Name)
		}

		result := fn.Call(toolCall.Function.Arguments)

		resultMessage := Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: toolCall.ID,
		}

		c.req.Messages = append(c.req.Messages, resultMessage)
	}
	return nil
}

// Subset of JSON Schema
type ParamDef struct {
	Type               string                 `json:"type"`
	Description        string                 `json:"description,omitempty"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
	Items              any                    `json:"items,omitempty"`
	Required           []string               `json:"required,omitempty"`
	AdditionProperties bool                   `json:"additionalProperties"`
}

type CallableFunction interface {
	Call(args string) string
	GetName() string
	GetDescription() string
	GetParameters() ParamDef
}
