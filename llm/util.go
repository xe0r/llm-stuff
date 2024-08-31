package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func GetToken() (string, error) {
	if token := os.Getenv("OPENROUTER_TOKEN"); token != "" {
		return token, nil
	}

	if token, err := secureGetToken(); err == nil {
		return token, nil
	}

	for _, filename := range []string{os.ExpandEnv("$HOME/.openrouter_token"), ".openrouter_token", ".token"} {
		if content, err := os.ReadFile(filename); err == nil {
			return strings.TrimSpace(string(content)), nil
		}
	}
	return "", fmt.Errorf("token not found")
}

func logMessage(msgs ...string) error {
	file, err := os.OpenFile("messages.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer file.Close()

	for i, msg := range msgs {
		msg = strings.TrimSpace(msg)
		if i > 0 {
			msg = " " + msg
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &jsonData); err == nil {
			var jsonFormatted bytes.Buffer
			if err := json.Indent(&jsonFormatted, []byte(msg), "", "  "); err == nil {
				msg = "\n" + jsonFormatted.String() + "\n"
			}
		}

		if i == len(msgs)-1 {
			msg += "\n\n"
		}
		if _, err := file.Write([]byte(msg)); err != nil {
			return err
		}
	}

	return nil
}
