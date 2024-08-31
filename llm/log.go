package llm

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
)

type Logger interface {
	Log(args ...string)
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

var DefaultLogger Logger = &defaultLogger{}

type defaultLogger struct{}

func (l *defaultLogger) Log(args ...string) {
	logMessage(args...)
}
