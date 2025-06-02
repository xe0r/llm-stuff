package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xe0r/llm-stuff/llm"
)

func run() error {
	token, err := llm.GetToken()
	if err != nil {
		return err
	}

	client := llm.NewChatClientWithType[string](token, nil)

	client.SetLogger(llm.DefaultLogger)

	//client.SetModel("mistralai/mistral-nemo")

	client.SetModel("openai/gpt-4o-mini")
	//client.SetModel("meta-llama/llama-3-70b-instruct")
	//client.SetModel("mistralai/mistral-7b-instruct")
	//client.SetModel("google/gemini-flash-1.5")

	client.AddMessage("system", `You are friendly chatbot.`)
	client.AddMessage("user", "Hello")

	stdinReader := bufio.NewScanner(os.Stdin)
	for {
		_, err := client.GetResponseWithCB(func(chunk string) {
			fmt.Print(chunk)
		})
		fmt.Print("\n")
		if err != nil {
			return err
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
