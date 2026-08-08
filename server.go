package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Register 挂上日志平台的 HTTP 接口。
func (h *Hub) Register(mux *http.ServeMux) {
	// 接收一条或多条日志，body 是 {"source":"","lines":["...","..."]} 或纯文本按行。
	mux.HandleFunc("/api/log", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
			return
		}
		source := r.URL.Query().Get("source")
		if source == "" {
			source = "unknown"
		}

		ct := r.Header.Get("Content-Type")
		if strings.Contains(ct, "application/json") {
			var body struct {
				Source string   `json:"source"`
				Lines  []string `json:"lines"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeErr(w, "JSON 解析失败: "+err.Error(), http.StatusBadRequest)
				return
			}
			if body.Source != "" {
				source = body.Source
			}
			for _, ln := range body.Lines {
				h.Ingest(ln, source)
			}
		} else {
			// 纯文本：按行拆，每行一条日志。
			buf := make([]byte, 1<<20)
			n, _ := r.Body.Read(buf)
			text := string(buf[:n])
			for _, ln := range strings.Split(text, "\n") {
				ln = strings.TrimRight(ln, "\r")
				if strings.TrimSpace(ln) != "" {
					h.Ingest(ln, source)
				}
			}
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})

	// 查最近的日志
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, h.Entries())
	})

	// 查告警
	mux.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, h.Alerts())
	})

	// 统计
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, h.Stats())
	})
}

const dashboardHTML = `<!doctype html><html lang="zh"><head><meta charset="utf-8">
<title>logmon</title><style>body{font-family:system-ui,sans-serif;max-width:760px;margin:30px auto}
pre{background:#0f1117;color:#d6e2ff;padding:12px;border-radius:6px;height:320px;overflow:auto}
.a{color:#ff6b6b}.meta{color:#888}</style></head><body>
<h2>go-logmon</h2>
<div id="stats" class="meta"></div>
<h3>实时日志</h3><pre id="log"></pre>
<h3>告警</h3><pre id="alert" class="a"></pre>
<script>
async function tick(){
  const s=await (await fetch('/api/stats')).json();
  stats.textContent='总 ' + s.total + ' 条 / 错误 ' + s.error_count + ' 条 / 告警 ' + s.alerts + ' 条';
  const logs=await (await fetch('/api/logs')).json();
  log.textContent=logs.slice(-200).map(e=>'['+e.level+'] '+e.source+': '+e.message).join('\n');
  log.scrollTop=log.scrollHeight;
  const al=await (await fetch('/api/alerts')).json();
  alert.textContent=al.slice(-50).map(a=>a.rule+' -> '+a.entry.message).join('\n');
}
tick(); setInterval(tick,2000);
</script></body></html>`
