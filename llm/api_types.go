package llm

type Request struct {
	Messages          []Message      `json:"messages,omitempty"`
	Prompt            string         `json:"prompt,omitempty"`
	Model             string         `json:"model,omitempty"`
	ResponseFormat    ResponseFormat `json:"response_format,omitempty"`
	Stop              string         `json:"stop,omitempty"`
	Stream            bool           `json:"stream,omitempty"`
	MaxTokens         int            `json:"max_tokens,omitempty"`
	Temperature       float64        `json:"temperature,omitempty"`
	TopP              float64        `json:"top_p,omitempty"`
	TopK              int            `json:"top_k,omitempty"`
	FrequencyPenalty  float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty   float64        `json:"presence_penalty,omitempty"`
	RepetitionPenalty float64        `json:"repetition_penalty,omitempty"`
	Seed              int            `json:"seed,omitempty"`
	Tools             []Tool         `json:"tools,omitempty"`
	// String or ToolChoice
	ToolChoice any                 `json:"tool_choice,omitempty"`
	LogitBias  map[int]float64     `json:"logit_bias,omitempty"`
	Transforms []string            `json:"transforms,omitempty"`
	Models     []string            `json:"models,omitempty"`
	Route      string              `json:"route,omitempty"`
	Provider   ProviderPreferences `json:"provider,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ImageContentPart struct {
	Type     string   `json:"type"`
	ImageURL ImageURL `json:"image_url"`
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ContentPart struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageURL ImageURL `json:"image_url,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Refusal string `json:"refusal,omitempty"`
	Name    string `json:"name,omitempty"`

	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	ToolCallID string `json:"tool_call_id,omitempty"`
}

type FunctionDescription struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Parameters  any    `json:"parameters"`
}

type Tool struct {
	Type     string              `json:"type"`
	Function FunctionDescription `json:"function"`
}

type ToolChoice struct {
	Type     string              `json:"type"`
	Function FunctionDescription `json:"function,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ProviderPreferences struct {
	RequireParameters bool `json:"require_parameters"`
}

type Response struct {
	ID                string   `json:"id"`
	Model             string   `json:"model"`
	Object            string   `json:"object"`
	Created           int      `json:"created"`
	Choices           []Choice `json:"choices"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Usage             *Usage   `json:"usage"`

	Error ErrorDef
	Code  int `json:"code"`
}

type ToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ErrorDef struct {
	Message string `json:"message"`
}

type Choice struct {
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
	Index        int         `json:"index"`
	Delta        *Message    `json:"delta,omitempty"`
	Message      *Message    `json:"message,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
