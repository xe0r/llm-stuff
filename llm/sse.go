package llm

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

type SSEReader struct {
	reader *bufio.Reader

	lastEventID   string
	reconnectTime int
}

func NewSSEReader(reader io.Reader) *SSEReader {
	return &SSEReader{
		reader: bufio.NewReader(reader),
	}
}

func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
	event := &SSEEvent{}
	for {
		line, err := r.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		lineStr := strings.TrimSpace(string(line))

		if len(lineStr) == 0 {
			if event.Event != "" || event.Data != "" {
				event.ID = r.lastEventID

				event.Data, _ = strings.CutSuffix(event.Data, "\n")

				return event, nil
			}

			continue
		}

		if lineStr[0] == ':' {
			continue
		}

		var field, value string
		if strings.Contains(lineStr, ":") {
			field, value, _ = strings.Cut(lineStr, ":")
			value, _ = strings.CutPrefix(value, " ")
		} else {
			field = lineStr
			value = ""
		}

		switch field {
		case "event":
			event.Event = value
		case "data":
			event.Data += value + "\n"
		case "id":
			if !strings.Contains(value, "\x00") {
				r.lastEventID = value
			}
		case "retry":
			if retry, err := strconv.Atoi(value); err == nil {
				r.reconnectTime = retry
			}
		}
	}
}
