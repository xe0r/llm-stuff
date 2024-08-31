package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xe0r/llm-stuff/llm"
)

type Response struct {
	Message string                 `json:"message" desc:"The message to be sent to the user."`
	Context map[string]interface{} `json:"new_context,omitempty" desc:"The updated context."`
}

func run() error {
	token, err := llm.GetToken()
	if err != nil {
		return err
	}

	contextContent, _ := os.ReadFile("context.json")
	if contextContent == nil {
		contextContent = []byte(`{}`)
	}

	client := llm.NewChatClientWithType[Response](token, nil)

	client.SetLogger(llm.DefaultLogger)

	//client.SetModel("mistralai/mistral-nemo")

	client.SetModel("openai/gpt-4o-mini")
	//client.SetModel("meta-llama/llama-3-70b-instruct")
	//client.SetModel("mistralai/mistral-7b-instruct")
	//client.SetModel("google/gemini-flash-1.5")

	client.SetObjectResponse()

	client.AddMessage("system", `You are context-aware assitant. You hold a context, which is a JSON map that contains all the stuff you remember about the user.
	Every time you receive a message from the user, you should update the context with the information from the message.
	You respond with JSON without any extra text.
	Your response should contain the message to be sent in field "message" and the updated context in field "new_context".
	Your saved context from previous interactions: `+string(contextContent))
	client.AddMessage("user", "Hello")

	stdinReader := bufio.NewScanner(os.Stdin)
	for {
		chunkReader := llm.NewChunkReader()

		chunkReader.Enable()

		resp, err := client.GetResponse(chunkReader.Chan())
		chunkReader.Wait()
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", resp.Message)

		if resp.Context != nil {
			contextContent, _ = json.Marshal(resp.Context)

			if err := os.WriteFile("context.json", contextContent, 0644); err != nil {
				fmt.Printf("Failed to save context: %v\n", err)
			}
		}

		fmt.Print(">>> ")
		stdinReader.Scan()
		if err := stdinReader.Err(); err != nil {
			return err
		}

		line := stdinReader.Text()
		line = strings.TrimSpace(line)

		if line == "" || line == "exit" || line == "quit" {
			break
		}

		client.AddMessage("user", line)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
