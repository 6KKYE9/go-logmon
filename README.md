# go-logmon

一个轻量日志采集 + 监控告警平台，纯标准库。把散落在各处的日志收集起来，解析成结构化（时间/级别/来源/内容），按规则匹配触发告警，再给个网页实时看板和告警列表。适合把几个服务的日志汇到一个地方统一盯。

## 跑起来

```powershell
go run .
```

浏览器开 http://127.0.0.1:9300 看实时看板。

## 喂日志进去

支持两种方式：

**1. HTTP 推送（结构化，推荐）**

```powershell
curl -X POST http://127.0.0.1:9300/api/log `
  -H "Content-Type: application/json" `
  -d '{"source":"auth","lines":["INFO login ok","ERROR db connection failed","WARN slow query"]}'
```

**2. 纯文本按行推**

```powershell
curl -X POST "http://127.0.0.1:9300/api/log?source=web" `
  -H "Content-Type: text/plain" `
  --data-binary @app.log
```

多行文本会按行拆，每行一条日志。

## 查数据

- `GET /api/logs` 最近所有日志
- `GET /api/alerts` 所有告警
- `GET /api/stats` 简单统计（总量/错误数/告警数）

## 告警规则怎么配的

`main` 里默认挂了两条规则：

- `error-level`：日志级别达到 ERROR（含 FATAL）就告警
- `keyword-timeout`：消息里含 "timeout" 就告警

级别有轻重排序：TRACE < DEBUG < INFO < WARN < ERROR < FATAL，所以 ERROR 规则对 FATAL 也生效。要加自己的规则改 `main.go` 里的 `AddRule` 即可。

告警出口：配了 `-webhook` 地址就会往那 POST（演示版只打印意图不真发）；不配就直接打到控制台。

## 日志解析

一行原始日志会试着用正则抠出 `INFO/WARN/ERROR/...` 这种级别词，抠不到归 INFO。时间拿不到就用接收时刻。真要做字段级解析（比如从 Nginx/Java 日志里精确提取时间、traceId），得给每种格式写对应的解析器，这里只做了通用兜底。

## 没做的事

- 没做文件 tail（Windows 没有 tail，跨平台监听文件变化要用 fsnotify 这类库），目前靠 HTTP 推
- 没持久化，日志重启就没
- webhook 只打印没真发，邮件/钉钉也没接

## 测试

```powershell
go test ./...
```

级别识别、规则匹配、摄入触发告警、HTTP 推送+查询，都有用例。
