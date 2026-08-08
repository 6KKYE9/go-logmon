package main

import (
	"regexp"
	"strings"
	"time"
)

// Entry 是一条解析后的日志。
type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Source  string    `json:"source"`
	Message string    `json:"message"`
}

// commonLogRe 试着从一行里抠出 "时间 级别 内容" 这种常见格式。
// 级别匹配 INFO/WARN/ERROR/DEBUG（大小写不限），匹配不到就归到 INFO。
var commonLogRe = regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL)\b`)

// parseLine 把一行原始日志解析成结构化 Entry。
// 先按常见格式试着找级别，找不到就当 INFO。时间拿不到就用当前时刻。
func parseLine(line, source string) Entry {
	level := "INFO"
	if m := commonLogRe.FindString(line); m != "" {
		level = strings.ToUpper(m)
		if level == "WARNING" {
			level = "WARN"
		}
	}
	return Entry{
		Time:    time.Now(),
		Level:   level,
		Source:  source,
		Message: strings.TrimSpace(line),
	}
}

// Alerter 是告警规则：命中 level 或匹配关键词就告警。
type Rule struct {
	Name    string `json:"name"`
	Level   string `json:"level"`   // 达到这个级别（含）就触发，空表示不限级别
	Keyword string `json:"keyword"` // 消息里包含这个关键词就触发，空表示不按关键词
}

// match 判断一条日志是否命中规则。
func (r Rule) match(e Entry) bool {
	if r.Level != "" && levelRank(e.Level) >= levelRank(r.Level) {
		return true
	}
	if r.Keyword != "" && strings.Contains(strings.ToLower(e.Message), strings.ToLower(r.Keyword)) {
		return true
	}
	return false
}

// levelRank 给级别排个序，方便按阈值比较。数字越大越严重。
func levelRank(l string) int {
	switch strings.ToUpper(l) {
	case "TRACE":
		return 0
	case "DEBUG":
		return 1
	case "INFO":
		return 2
	case "WARN", "WARNING":
		return 3
	case "ERROR":
		return 4
	case "FATAL":
		return 5
	default:
		return 2
	}
}
