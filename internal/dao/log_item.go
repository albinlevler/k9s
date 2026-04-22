// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dao

import (
	"bytes"
)

// Log detail levels for cycling through log verbosity.
const (
	LogDetailCompact = 0 // timestamp + level + message
	LogDetailMedium  = 1 // timestamp + level + class + message
	LogDetailRaw     = 2 // raw log line (no filtering)
)

// LogChan represents a channel for logs.
type LogChan chan *LogItem

var ItemEOF = new(LogItem)

// LogItem represents a container log line.
type LogItem struct {
	Pod, Container  string
	SingleContainer bool
	Bytes           []byte
	IsError         bool
}

// NewLogItem returns a new item.
func NewLogItem(bb []byte) *LogItem {
	return &LogItem{
		Bytes: bb,
	}
}

// NewLogItemFromString returns a new item.
func NewLogItemFromString(s string) *LogItem {
	return &LogItem{
		Bytes: []byte(s),
	}
}

// ID returns pod and or container based id.
func (l *LogItem) ID() string {
	if l.Pod != "" {
		return l.Pod
	}
	return l.Container
}

// GetTimestamp fetch log lime timestamp
func (l *LogItem) GetTimestamp() string {
	index := bytes.Index(l.Bytes, []byte{' '})
	if index < 0 {
		return ""
	}
	return string(l.Bytes[:index])
}

// Info returns pod and container information.
func (l *LogItem) Info() string {
	return l.Pod + "::" + l.Container
}

// IsEmpty checks if the entry is empty.
func (l *LogItem) IsEmpty() bool {
	return len(l.Bytes) == 0
}

// Size returns the size of the item.
func (l *LogItem) Size() int {
	return 100 + len(l.Bytes) + len(l.Pod) + len(l.Container)
}

// Render returns a log line as string.
func (l *LogItem) Render(paint string, showTime bool, detailLevel int, bb *bytes.Buffer) {
	index := bytes.Index(l.Bytes, []byte{' '})
	if showTime && index > 0 {
		bb.WriteString("[gray::b]")
		bb.Write(l.Bytes[:index])
		bb.WriteString(" ")
		if l := 30 - len(l.Bytes[:index]); l > 0 {
			bb.Write(bytes.Repeat([]byte{' '}, l))
		}
		bb.WriteString("[-::-]")
	}

	var content []byte
	if index > 0 {
		content = l.Bytes[index+1:]
	} else {
		content = l.Bytes
	}

	if detailLevel < LogDetailRaw {
		if appTS, level, class, message, ok := parseAppLogContent(content); ok {
			bb.WriteString("[gray::]")
			bb.Write(appTS)
			bb.WriteString("[-::] ")
			bb.WriteString(levelColorTag(level))
			bb.Write(level)
			bb.WriteString("[-::] ")
			if detailLevel == LogDetailMedium {
				bb.WriteString("[yellow::b]")
				bb.Write(class)
				bb.WriteString("[-::] ")
			}
			bb.Write(message)
			return
		}
	}

	if l.Pod != "" {
		bb.WriteString("[" + paint + "::]" + l.Pod)
	}
	if !l.SingleContainer && l.Container != "" {
		if l.Pod != "" {
			bb.WriteString(" ")
		}
		bb.WriteString("[" + paint + "::b]" + l.Container + "[-::-] ")
	} else if l.Pod != "" {
		bb.WriteString("[-::] ")
	}
	bb.Write(content)
}

// parseAppLogContent parses a log line in the format:
// DATE TIME LEVEL CLASS [THREAD] [meta=...] [...] [...] MESSAGE
// Returns (appTimestamp, level, class, message, ok).
func parseAppLogContent(content []byte) (appTS, level, class, message []byte, ok bool) {
	parts := bytes.SplitN(content, []byte{' '}, 4)
	if len(parts) < 4 {
		return nil, nil, nil, content, false
	}
	if !isKnownLogLevel(parts[2]) {
		return nil, nil, nil, content, false
	}

	appTS = content[:len(parts[0])+1+len(parts[1])]
	level = parts[2]
	rest := parts[3]

	spaceIdx := bytes.IndexByte(rest, ' ')
	if spaceIdx < 0 {
		return appTS, level, rest, []byte{}, true
	}
	class = rest[:spaceIdx]
	rest = rest[spaceIdx+1:]

	for len(rest) > 0 && rest[0] == '[' {
		end := bytes.IndexByte(rest, ']')
		if end < 0 {
			break
		}
		rest = rest[end+1:]
		if len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
	}
	return appTS, level, class, rest, true
}

func isKnownLogLevel(b []byte) bool {
	switch len(b) {
	case 4:
		return bytes.Equal(b, []byte("INFO")) || bytes.Equal(b, []byte("WARN"))
	case 5:
		return bytes.Equal(b, []byte("ERROR")) || bytes.Equal(b, []byte("DEBUG")) || bytes.Equal(b, []byte("TRACE"))
	}
	return false
}

func levelColorTag(level []byte) string {
	switch string(level) {
	case "ERROR":
		return "[red::b]"
	case "WARN":
		return "[yellow::]"
	case "DEBUG":
		return "[blue::]"
	case "TRACE":
		return "[gray::]"
	default:
		return "[green::]"
	}
}
