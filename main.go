package main

import (
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	lastText  string
	savedAt   time.Time
	mu        sync.RWMutex
	password  string
)

func main() {
	password = os.Getenv("PASSWORD")
	if password == "" {
		panic("PASSWORD env not set")
	}

	http.HandleFunc("/", handleMain)
	http.HandleFunc("/raw", handleRaw)
	http.ListenAndServe(":8080", nil)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	msg := ""
	if r.Method == "POST" {
		if r.FormValue("password") != password {
			msg = "wrong password"
		} else {
			mu.Lock()
			lastText = r.FormValue("text")
			savedAt = time.Now()
			mu.Unlock()
			msg = "saved"
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>txt</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#000;color:#fff;font-family:monospace;padding:20px;max-width:600px;margin:0 auto}
textarea{width:100%;height:200px;background:#111;color:#fff;border:1px solid #333;padding:10px;font-family:monospace;resize:vertical}
input{background:#111;color:#fff;border:1px solid #333;padding:10px;font-family:monospace;width:100%;margin-top:10px}
button{background:#fff;color:#000;border:none;padding:10px 20px;cursor:pointer;font-family:monospace;margin-top:10px}
button:hover{background:#aaa}
.msg{margin-bottom:10px;color:#888}
</style>
</head>
<body>
<form method="POST">
<div class="msg">` + msg + `</div>
<textarea name="text" placeholder="type anything"></textarea>
<input type="password" name="password" placeholder="password">
<button type="submit">send</button>
</form>
</body>
</html>`))
}

func handleRaw(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	text := lastText
	ts := savedAt
	mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if text == "" || time.Since(ts) > 5*time.Minute {
		w.WriteHeader(http.StatusGone)
		w.Write([]byte("expired"))
		return
	}

	w.Write([]byte(text))
}
