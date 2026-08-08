package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":9300", "监听地址")
	webhook := flag.String("webhook", "", "告警 webhook 地址（空就不发，只打印）")
	keep := flag.Int("keep", 1000, "最多保留多少条日志")
	flag.Parse()

	hub := newHub(*keep, *webhook)
	// 默认规则：ERROR/FATAL 必告警，出现 "timeout" 关键词也告警。
	hub.AddRule(Rule{Name: "error-level", Level: "ERROR"})
	hub.AddRule(Rule{Name: "keyword-timeout", Keyword: "timeout"})

	mux := http.NewServeMux()
	hub.Register(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
	})

	log.Printf("logmon 启动 %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
