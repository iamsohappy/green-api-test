package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", pageHandler)
	mux.HandleFunc("/api/getSettings", proxyHandler("getSettings"))
	mux.HandleFunc("/api/getStateInstance", proxyHandler("getStateInstance"))
	mux.HandleFunc("/api/sendMessage", proxyHandler("sendMessage"))
	mux.HandleFunc("/api/sendFileByUrl", proxyHandler("sendFileByUrl"))

	fmt.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, pageHTML)
}

func proxyHandler(method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDInstance       string `json:"idInstance"`
			ApiTokenInstance string `json:"apiTokenInstance"`
			ChatID           string `json:"chatId"`
			Message          string `json:"message"`
			URLFile          string `json:"urlFile"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		resp, err := callGREENAPI(method, req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.WriteString(w, resp.Body)
	}
}

type apiResp struct {
	StatusCode int
	Body       string
}

func callGREENAPI(method string, req struct {
	IDInstance       string `json:"idInstance"`
	ApiTokenInstance string `json:"apiTokenInstance"`
	ChatID           string `json:"chatId"`
	Message          string `json:"message"`
	URLFile          string `json:"urlFile"`
}) (*apiResp, error) {
	if req.IDInstance == "" || req.ApiTokenInstance == "" {
		return nil, fmt.Errorf("idInstance and apiTokenInstance are required")
	}

	url := fmt.Sprintf("https://api.green-api.com/waInstance%s/%s/%s", strings.TrimSpace(req.IDInstance), method, req.ApiTokenInstance)

	var body io.Reader
	switch method {
	case "getSettings", "getStateInstance":
		body = nil
	case "sendMessage":
		payload := map[string]string{
			"chatId":  normalizeChatID(req.ChatID),
			"message": req.Message,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "sendFileByUrl":
		payload := map[string]string{
			"chatId":   normalizeChatID(req.ChatID),
			"urlFile":  req.URLFile,
			"fileName": "file",
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return &apiResp{StatusCode: res.StatusCode, Body: string(b)}, nil
}

func normalizeChatID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.Contains(v, "@") {
		return v
	}
	return strings.TrimLeft(v, "+") + "@c.us"
}

var pageHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>GREEN-API test task</title>
<style>
body{font-family:Arial,sans-serif;margin:0;padding:24px;background:#fff;color:#222}
.wrap{max-width:1200px;margin:0 auto;border:1px solid #bbb;min-height:92vh;padding:20px}
.topbar{height:44px;border-bottom:1px solid #aaa;margin-bottom:30px;display:flex;align-items:center;gap:10px}
.dot{width:14px;height:14px;border-radius:50%;border:1px solid #999}
.grid{display:grid;grid-template-columns:360px 1fr;gap:28px}
.left{display:flex;flex-direction:column;gap:18px}
input,textarea,button{width:100%;box-sizing:border-box;font-size:14px;padding:12px;border:1px solid #aaa;border-radius:3px}
textarea{min-height:88px;resize:vertical}
button{background:#f6f6f6;cursor:pointer}button:hover{background:#efefef}
.response{min-height:700px}.label{font-size:18px;margin-bottom:8px}.footer{margin-top:40px;text-align:right;color:#0a7a2f;font-weight:700}
</style>
</head>
<body>
<div class="wrap">
<div class="topbar"><div class="dot"></div><div class="dot"></div><div class="dot"></div></div>
<div class="grid">
<div class="left">
<input id="idInstance" placeholder="idInstance" />
<input id="apiTokenInstance" placeholder="ApiTokenInstance" />
<button onclick="callMethod('getSettings')">getSettings</button>
<button onclick="callMethod('getStateInstance')">getStateInstance</button>
<input id="chatId" placeholder="77771234567" />
<textarea id="message" placeholder="Hello World!"></textarea>
<button onclick="callMethod('sendMessage')">sendMessage</button>
<input id="fileChatId" placeholder="77771234567" />
<input id="urlFile" placeholder="https://my.site.com/my/horse.png" />
<button onclick="callMethod('sendFileByUrl')">sendFileByUrl</button>
</div>
<div>
<div class="label">Ответ:</div>
<textarea id="response" class="response" readonly></textarea>
</div>
</div>
<div class="footer">GREEN API © 2025</div>
</div>
<script>
async function callMethod(method){
const payload={idInstance:document.getElementById('idInstance').value.trim(),apiTokenInstance:document.getElementById('apiTokenInstance').value.trim()}
if(method==='sendMessage'){payload.chatId=document.getElementById('chatId').value.trim();payload.message=document.getElementById('message').value}
if(method==='sendFileByUrl'){payload.chatId=document.getElementById('fileChatId').value.trim();payload.urlFile=document.getElementById('urlFile').value.trim()}
const response=document.getElementById('response')
response.value='Loading...'
try{
const res=await fetch('/api/'+method,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
const text=await res.text()
try{response.value=JSON.stringify(JSON.parse(text),null,2)}catch{response.value=text}
}catch(e){response.value=String(e)}
}
</script>
</body>
</html>`
