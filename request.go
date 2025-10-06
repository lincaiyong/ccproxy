package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type SystemMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Metadata struct {
	UserID string `json:"user_id"`
}

type InputSchema struct {
	AdditionalProperties bool     `json:"additionalProperties"`
	Properties           any      `json:"properties"`
	Required             []string `json:"required"`
	Type                 string   `json:"type"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

type Request struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens"`
	Messages      []Message       `json:"messages"`
	System        []SystemMessage `json:"system"`
	StopSequences interface{}     `json:"stop_sequences"`
	Stream        bool            `json:"stream"`
	Temperature   float64         `json:"temperature"`
	TopP          interface{}     `json:"top_p"`
	TopK          interface{}     `json:"top_k"`
	Metadata      Metadata        `json:"metadata"`
	Tools         []Tool          `json:"tools"`
	ToolChoice    interface{}     `json:"tool_choice"`
	Thinking      interface{}     `json:"thinking"`
}

func writeTagNl(sb strings.Builder, tag, s string) strings.Builder {
	s = strings.TrimSpace(s)
	sb.WriteString(fmt.Sprintf("<%s>\n", tag))
	for _, line := range strings.Split(s, "\n") {
		sb.WriteString(fmt.Sprintf("  %s\n", line))
	}
	sb.WriteString(fmt.Sprintf("</%s>\n", tag))
	return sb
}

func (req *Request) Compose() string {
	var sb strings.Builder
	for _, m := range req.System {
		sb = writeTagNl(sb, "system", m.Text)
	}
	disabledTools := map[string]bool{
		"Task":                     true,
		"Bash":                     false,
		"Glob":                     false,
		"Grep":                     false,
		"LS":                       false,
		"ExitPlanMode":             true,
		"Read":                     false,
		"Edit":                     false,
		"MultiEdit":                true,
		"Write":                    false,
		"NotebookEdit":             true,
		"WebFetch":                 true,
		"TodoWrite":                false,
		"WebSearch":                false,
		"BashOutput":               true,
		"KillBash":                 true,
		"mcp__ide__getDiagnostics": true,
	}
	if len(req.Tools) > 0 {
		tools := make([]Tool, 0)
		for _, tool := range req.Tools {
			if disabledTools[tool.Name] {
				continue
			}
			tools = append(tools, tool)
		}
		b, _ := json.MarshalIndent(tools, "", "  ")
		content := fmt.Sprintf(`USE TOOL
--------
Specify what tool to use and the required arguments in <use></use> block.
- Place tool name in "tool" attribute.
- Place tool arguments between <use> and </use>
- The tool arguments MUST be a valid JSON object that can be validated against the tool's input JSON schema.
ALWAYS check the existing facts before calling the tool, DO NOT call tools repeatedly.
Once you respond with </use>, you STOP.

<examples>
	<good_example>
		<use tool="Edit">
		{
			"file_path": "/path/to/main.py",
			"old_string": "class Snippet:\n    def __init__(self, file_path, line_no, lines):",
			"new_string": "class Snippet:\n    def __init__(self, file_path, line_no, lines, context_range=4):"
		}
		</use>
	</good_example>

	<bad_example>
		<use>
		{
			"tool": "Edit",
			"file_path": "/path/to/main.py",
			"old_string": "class Snippet:\n    def __init__(self, file_path, line_no, lines):",
			"new_string": "class Snippet:\n    def __init__(self, file_path, line_no, lines, context_range=4):"
		}
		</use>
		<reasoning>
			The tool name should be placed in "tool" attribute!
		</reasoning>
	</bad_example>

	<bad_example>
		<use tool="Read">
			<file_path>/path/to/main.go</file_path>
			<offset>116</offset>
			<limit>110</limit>
		</use>
		<reasoning>
			The tool arguments MUST be a valid JSON object.
		</reasoning>
	</bad_example>
</examples>

AVAILABLE TOOLS
---------------
%s`, string(b))
		sb = writeTagNl(sb, "tools", content)
	}
	for _, m := range req.Messages {
		if s, ok := m.Content.(string); ok {
			sb = writeTagNl(sb, m.Role, s)
		} else if lst, ok2 := m.Content.([]any); ok2 {
			for _, el := range lst {
				if mm, ok3 := el.(map[string]interface{}); ok3 {
					type_ := mm["type"].(string)
					if type_ == "text" {
						ss, _ := mm["text"].(string)
						sb = writeTagNl(sb, m.Role, ss)
					} else {
						b, _ := json.MarshalIndent(mm, "", "  ")
						sb = writeTagNl(sb, m.Role, string(b))
					}
				}
			}
		} else {
			b, _ := json.MarshalIndent(m.Content, "", "  ")
			sb = writeTagNl(sb, m.Role, string(b))
		}
	}
	return sb.String()
}
