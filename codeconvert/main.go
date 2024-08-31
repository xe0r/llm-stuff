package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/xe0r/llm-stuff/llm"
)

func doit(inputName, outputName, language string) error {
	var input io.ReadCloser
	var output io.WriteCloser

	if inputName != "" && inputName != "-" {
		var err error
		input, err = os.Open(inputName)
		if err != nil {
			return err
		}
	} else {
		input = os.Stdin
	}
	content, err := io.ReadAll(input)
	_ = input.Close()
	if err != nil {
		return err
	}

	token, err := llm.GetToken()
	if err != nil {
		return err
	}

	if outputName != "" && outputName != "-" {
		var err error
		output, err = os.Create(outputName)
		if err != nil {
			return err
		}
	} else {
		output = os.Stdout
	}
	defer func() { _ = output.Close() }()

	client := llm.NewChatClient(token, nil)

	client.SetModel("openai/gpt-4o-mini")
	//client.SetModel("meta-llama/llama-3-70b-instruct")
	//client.SetModel("mistralai/mistral-7b-instruct")
	//client.SetModel("google/gemini-flash-1.5")

	client.AddMessage("system", fmt.Sprintf("You are code conversion tool. You convert code from any language to %s. You respond with the converted code without any comments or markdown.", language))
	client.AddMessage("user", string(content))

	chunkChan := make(chan string)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for chunk := range chunkChan {
			fmt.Fprint(output, chunk)
		}
		fmt.Fprintln(output)
	}()

	resp, err := client.GetResponse(chunkChan)
	<-doneChan
	if err != nil {
		return err
	}

	if chunkChan == nil {
		fmt.Fprintln(output, resp)
	}
	return nil
}

func main() {
	var (
		inputName  string
		outputName string
		language   string
	)

	cmd := &cobra.Command{
		Use:   "codeconvert",
		Short: "Convert code from one language to another",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doit(inputName, outputName, language)
		},
	}
	cmd.Flags().StringVarP(&inputName, "input", "i", "", "Input file name")
	cmd.Flags().StringVarP(&outputName, "output", "o", "", "Output file name")
	cmd.Flags().StringVarP(&language, "language", "l", "Go", "Language to convert to")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
