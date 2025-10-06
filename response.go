package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"strings"
	"time"
)

const EventTypeContentBlockDelta = "content_block_delta"
const EventTypeContentBlockStart = "content_block_start"
const EventTypeContentBlockStop = "content_block_stop"
const EventTypeMessageDelta = "message_delta"
const EventTypeMessageStart = "message_start"
const EventTypeMessageStop = "message_stop"
const EventTypePing = "ping"

type Response struct {
	index int
}

func (r *Response) IncIndex() {
	r.index++
}

func (r *Response) Write(c *gin.Context, event *Event) {
	event.Index = r.index
	data := event.Data()
	//log.InfoLog("chunk: %s", data)
	_, err := c.Writer.Write([]byte(data))
	if err != nil {
		panic(err)
	}
	c.Writer.Flush()
}

type Event struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Text  string `json:"text"`
	Tool  string `json:"tool"`
	Args  string `json:"args"`
}

func NewContentBlockDeltaEvent(text string) *Event {
	return &Event{Type: EventTypeContentBlockDelta, Text: text}
}

func NewContentBlockDeltaEventWithTool(tool string, args string) *Event {
	return &Event{Type: EventTypeContentBlockDelta, Tool: tool, Args: args}
}

func NewContentBlockStartEvent() *Event {
	return &Event{Type: EventTypeContentBlockStart}
}

func NewContentBlockStartEventWithTool(tool string) *Event {
	return &Event{Type: EventTypeContentBlockStart, Tool: tool}
}

func NewContentBlockStopEvent() *Event {
	return &Event{Type: EventTypeContentBlockStop}
}

func NewContentBlockStopEventWithTool(tool string) *Event {
	return &Event{Type: EventTypeContentBlockStop, Tool: tool}
}

func NewMessageStartEvent() *Event {
	return &Event{Type: EventTypeMessageStart}
}

func NewMessageDeltaEventWithTool(tool string) *Event {
	return &Event{Type: EventTypeMessageDelta, Tool: tool}
}

func NewMessageStopEvent() *Event {
	return &Event{Type: EventTypeMessageStop}
}

func NewPingEvent() *Event {
	return &Event{Type: EventTypePing}
}

func uuid24() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 24
	result := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}

func (e *Event) Data() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("event: %s\n", e.Type))
	sb.WriteString(fmt.Sprintf(`data: {"type": "%s"`, e.Type))
	switch e.Type {
	case EventTypeContentBlockDelta:
		if e.Tool == "" {
			b, _ := json.Marshal(e.Text)
			sb.WriteString(fmt.Sprintf(`, "index": %d, "delta": {"type": "text_delta", "text": %s}`, e.Index, string(b)))
		} else {
			b, _ := json.Marshal(e.Args)
			sb.WriteString(fmt.Sprintf(`, "index": %d, "delta": {"type": "input_json_delta", "partial_json": %s}`, e.Index, string(b)))
		}
	case EventTypeContentBlockStart:
		if e.Tool == "" {
			sb.WriteString(fmt.Sprintf(`, "index": %d, "content_block": {"type": "text", "text": ""}`, e.Index))
		} else {
			sb.WriteString(fmt.Sprintf(`, "index": %d, "content_block": {"type": "tool_use", "id": "call_%s", "name": "%s", "input": {}}`, e.Index, uuid24(), e.Tool))
		}
	case EventTypeContentBlockStop:
		sb.WriteString(fmt.Sprintf(`, "index": %d`, e.Index))
	case EventTypeMessageDelta:
		if e.Tool == "" {
			sb.WriteString(`, "delta": {"stop_reason": "end_turn", "stop_sequence": null}, "usage": {"input_tokens": 89, "output_tokens": 11, "cache_read_input_tokens": 11392}`)
		} else {
			sb.WriteString(`, "delta": {"stop_reason": "tool_use", "stop_sequence": null}, "usage": {"input_tokens": 89, "output_tokens": 11, "cache_read_input_tokens": 11392}`)
		}
	case EventTypeMessageStart:
		sb.WriteString(fmt.Sprintf(`, "message": {"id": "msg_%s", "type": "message", "role": "assistant", "model": "claude-sonnet-4-20250514", "content": [], "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 0, "output_tokens": 0}}`, uuid24()))
	default:
	}
	sb.WriteString("}\n\n")
	return sb.String()
}
