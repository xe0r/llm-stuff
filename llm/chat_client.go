package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

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
	switch ty.Kind() {
	case reflect.String:
		req.ResponseFormat.Type = "text"
	case reflect.Map:
		req.ResponseFormat.Type = "json_object"
	default:
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

func (c *ChatClient[T]) SetLogger(logger Logger) {
	c.client.SetLogger(logger)
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
			doneChan := make(chan struct{})

			go func() {
				defer close(doneChan)
				for chunk := range subChunkChan {
					if len(chunk.Choices) > 0 {
						chunkChan <- chunk.Choices[0].Delta.Content
					}
				}
			}()
			resp, err = c.client.SendStreamRequest(c.req, subChunkChan)
			<-doneChan
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

		switch strings.ToLower(choice.FinishReason) {
		case "stop", "":
			// It seems that some models don't send finish reason, at least in the stream mode			
			return c.convertResult(choice.Message.Content)
		case "tool_calls":
			err := c.handleToolCalls(choice.Message.ToolCalls)
			if err != nil {
				return result, err
			}
		default:
			return result, fmt.Errorf("unknown finish reason %s", choice.FinishReason)
		}
	}
}

func (c *ChatClient[T]) convertResult(content string) (T, error) {
	result := *new(T)
	tv := reflect.ValueOf(&result).Elem()
	if tv.Kind() == reflect.String {
		tv.SetString(content)
		return result, nil
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return result, err
	}
	return result, nil
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
