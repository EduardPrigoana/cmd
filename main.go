package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	htmlPrefix = `<!DOCTYPE html>
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
<div class="msg">`

	htmlMid = `</div>
<textarea name="text" placeholder="type anything"></textarea>
`

	htmlSuffix = `<button type="submit">send</button>
</form>
</body>
</html>`

	pwField        = `<input type="password" name="password" placeholder="password">`
	expiration     = 5 * time.Minute
	cookieDuration = 30 * 24 * time.Hour
)

type store struct {
	sync.RWMutex
	text    string
	savedAt time.Time
}

func (s *store) Set(text string) {
	s.Lock()
	s.text = text
	s.savedAt = time.Now()
	s.Unlock()
}

func (s *store) Get() (string, time.Time) {
	s.RLock()
	t, ts := s.text, s.savedAt
	s.RUnlock()
	return t, ts
}

var (
	data      store
	password  string
	authToken []byte
)

func main() {
	password = os.Getenv("PASSWORD")
	if password == "" {
		password = "PASSWORD"
	}

	h := sha256.Sum256([]byte(password))
	authToken = []byte(hex.EncodeToString(h[:]))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleGet)
	mux.HandleFunc("POST /{$}", handlePost)
	mux.HandleFunc("GET /raw", handleRaw)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	srv.ListenAndServe()
}

func isAuthed(r *http.Request) bool {
	c, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), authToken) == 1
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	renderPage(w, r, "")
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	if !isAuthed(r) {
		if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(password)) != 1 {
			renderPage(w, r, "wrong password")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    string(authToken),
			Expires:  time.Now().Add(cookieDuration),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
	}

	data.Set(r.FormValue("text"))
	renderPage(w, r, "saved")
}

func renderPage(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	io.WriteString(w, htmlPrefix)
	io.WriteString(w, msg)
	io.WriteString(w, htmlMid)
	if !isAuthed(r) {
		io.WriteString(w, pwField)
	}
	io.WriteString(w, htmlSuffix)
}

func handleRaw(w http.ResponseWriter, r *http.Request) {
	text, ts := data.Get()

	if text == "" || time.Since(ts) > expiration {
		http.Error(w, "expired", http.StatusGone)
		return
	}

	io.WriteString(w, text)
}
