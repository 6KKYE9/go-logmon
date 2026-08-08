package main

import (
	"fmt"
	"sync"
	"time"
)

// Hub 是日志平台的中心：收日志、存最近 N 条、按规则匹配产生告警、
// 把告警通过 Webhook 发出去（演示用打印 + 内存留存）。
type Hub struct {
	mu      sync.Mutex
	entries []Entry
	maxKeep int
	rules   []Rule
	alerts  []Alert
	webhook string // 告警 webhook 地址，空就不发
}

// Alert 是一条触发的告警记录。
type Alert struct {
	Time  time.Time `json:"time"`
	Rule  string    `json:"rule"`
	Entry Entry     `json:"entry"`
}

func newHub(maxKeep int, webhook string) *Hub {
	if maxKeep <= 0 {
		maxKeep = 1000
	}
	return &Hub{maxKeep: maxKeep, webhook: webhook}
}

// AddRule 加一条告警规则。
func (h *Hub) AddRule(r Rule) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rules = append(h.rules, r)
}

// Ingest 摄入一行原始日志：解析、留存、匹配规则、必要时告警。
func (h *Hub) Ingest(line, source string) {
	e := parseLine(line, source)

	h.mu.Lock()
	h.entries = append(h.entries, e)
	if len(h.entries) > h.maxKeep {
		h.entries = h.entries[len(h.entries)-h.maxKeep:]
	}

	var fired []Alert
	for _, r := range h.rules {
		if r.match(e) {
			a := Alert{Time: time.Now(), Rule: r.Name, Entry: e}
			h.alerts = append(h.alerts, a)
			if len(h.alerts) > 500 {
				h.alerts = h.alerts[len(h.alerts)-500:]
			}
			fired = append(fired, a)
		}
	}
	h.mu.Unlock()

	// 发 webhook 放锁外，避免阻塞摄入主流程。
	for _, a := range fired {
		h.dispatch(a)
	}
}

// dispatch 把告警发出去。没配 webhook 就只打印，模拟发送。
func (h *Hub) dispatch(a Alert) {
	msg := fmt.Sprintf("[ALERT] %s: %s %s", a.Rule, a.Entry.Level, a.Entry.Message)
	if h.webhook == "" {
		fmt.Println(msg)
		return
	}
	// 真发 webhook 会在这里 POST，演示版省略网络调用，只记录意图。
	fmt.Printf("%s -> (webhook %s)\n", msg, h.webhook)
}

func (h *Hub) Entries() []Entry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Entry, len(h.entries))
	copy(out, h.entries)
	return out
}

func (h *Hub) Alerts() []Alert {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Alert, len(h.alerts))
	copy(out, h.alerts)
	return out
}

// Stats 给看板用的简单统计。
func (h *Hub) Stats() map[string]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := map[string]int{"total": len(h.entries), "alerts": len(h.alerts)}
	byLevel := map[string]int{}
	for _, e := range h.entries {
		byLevel[e.Level]++
	}
	s["by_level"] = 0
	for _, n := range byLevel {
		s["by_level"] += n
	}
	s["error_count"] = byLevel["ERROR"] + byLevel["FATAL"]
	return s
}
