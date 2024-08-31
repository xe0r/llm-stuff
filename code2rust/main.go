package main

import (
	"fmt"
	"io"
	"os"

	"github.com/xe0r/llm-stuff/llm"
)

func run() error {
	token, err := llm.GetToken()
	if err != nil {
		return err
	}

	client := llm.NewChatClient(token, nil)

	client.SetModel("openai/gpt-4o-mini")
	//client.SetModel("meta-llama/llama-3-70b-instruct")
	//client.SetModel("mistralai/mistral-7b-instruct")
	//client.SetModel("google/gemini-flash-1.5")

	stdinContent, _ := io.ReadAll(os.Stdin)

	client.AddMessage("system", "You are code conversion tool. You convert code from any language to Rust. You respond with the converted code without any comments or markdown.")
	client.AddMessage("user", string(stdinContent))

	chunkChan := make(chan string)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for chunk := range chunkChan {
			fmt.Print(chunk)
		}
		fmt.Println()
	}()

	resp, err := client.GetResponse(chunkChan)
	<-doneChan
	if err != nil {
		return err
	}

	if chunkChan == nil {
		fmt.Println(resp)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
