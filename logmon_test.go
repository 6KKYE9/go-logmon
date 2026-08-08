package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	if e := parseLine("2026-01-01 INFO hello", "a"); e.Level != "INFO" {
		t.Fatalf("应识别 INFO, 得到 %s", e.Level)
	}
	if e := parseLine("something ERROR boom", "a"); e.Level != "ERROR" {
		t.Fatalf("应识别 ERROR, 得到 %s", e.Level)
	}
	if e := parseLine("plain line no level", "a"); e.Level != "INFO" {
		t.Fatalf("无级别应默认 INFO, 得到 %s", e.Level)
	}
}

func TestRuleMatch(t *testing.T) {
	r := Rule{Name: "x", Level: "ERROR"}
	if !r.match(Entry{Level: "ERROR"}) {
		t.Fatal("ERROR 应匹配 ERROR 规则")
	}
	if !r.match(Entry{Level: "FATAL"}) {
		t.Fatal("FATAL 比 ERROR 严重，应匹配")
	}
	if r.match(Entry{Level: "INFO"}) {
		t.Fatal("INFO 不应匹配 ERROR 规则")
	}
	kw := Rule{Name: "y", Keyword: "timeout"}
	if !kw.match(Entry{Message: "connection timeout"}) {
		t.Fatal("含 timeout 应匹配")
	}
}

func TestIngestFiresAlert(t *testing.T) {
	hub := newHub(100, "")
	hub.AddRule(Rule{Name: "err", Level: "ERROR"})
	hub.Ingest("INFO normal", "svc")
	if len(hub.Alerts()) != 0 {
		t.Fatal("普通 INFO 不应告警")
	}
	hub.Ingest("ERROR something broke", "svc")
	if len(hub.Alerts()) != 1 {
		t.Fatalf("ERROR 应产生 1 条告警, 得到 %d", len(hub.Alerts()))
	}
	if len(hub.Entries()) != 2 {
		t.Fatalf("应留存 2 条日志, 得到 %d", len(hub.Entries()))
	}
}

func TestHTTPIngestAndQuery(t *testing.T) {
	hub := newHub(100, "")
	hub.AddRule(Rule{Name: "err", Level: "ERROR"})
	mux := http.NewServeMux()
	hub.Register(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 推一批日志（JSON 格式）
	body := `{"source":"auth","lines":["INFO login ok","ERROR db connection failed","WARN slow query"]}`
	resp, _ := http.Post(ts.URL+"/api/log", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("推送状态码 %d", resp.StatusCode)
	}

	logs, _ := http.Get(ts.URL + "/api/logs")
	defer logs.Body.Close()
	var entries []Entry
	json.NewDecoder(logs.Body).Decode(&entries)
	if len(entries) != 3 {
		t.Fatalf("应收到 3 条, 得到 %d", len(entries))
	}

	al, _ := http.Get(ts.URL + "/api/alerts")
	defer al.Body.Close()
	var alerts []Alert
	json.NewDecoder(al.Body).Decode(&alerts)
	if len(alerts) != 1 {
		t.Fatalf("应 1 条告警, 得到 %d", len(alerts))
	}
}
