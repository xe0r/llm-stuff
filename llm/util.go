package llm

import (
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
