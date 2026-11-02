package main

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"sync"
)

// Учебная учётная запись намеренно хранится в коде.
// Слабая проверка пароля и отсутствие защиты от перебора — часть задания.
const (
	trainingLogin    = "admin"
	trainingPassword = "111111"
	trainingFlag     = "CTF{passwords_need_rate_limits}"
)

const sessionCookie = "training_session"

var (
	sessions   = make(map[string]bool)
	sessionsMu sync.RWMutex

	loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Training Login</title><style>
body{font-family:system-ui,sans-serif;background:#101820;color:#eaf2f8;display:grid;place-items:center;min-height:100vh;margin:0}
main{background:#182733;border:1px solid #2f495b;border-radius:12px;padding:2rem;width:min(90vw,360px)}
h1{margin-top:0}label{display:block;margin:1rem 0 .35rem}input,button{box-sizing:border-box;width:100%;padding:.7rem;border-radius:7px;border:1px solid #547084}
button{margin-top:1.4rem;background:#53c7e8;color:#071117;font-weight:700;cursor:pointer}.error{color:#ff9b9b;min-height:1.4rem}.note{color:#acc1ce;font-size:.9rem}
</style></head><body><main><h1>Личный кабинет</h1><p class="note">Учебный локальный стенд.</p>
<form method="post" action="/login"><label for="login">Логин</label><input id="login" name="login" required autofocus>
<label for="password">Пароль</label><input id="password" type="password" name="password" required>
<p class="error">{{.Error}}</p><button type="submit">Войти</button></form></main></body></html>`))

	profilePage = template.Must(template.New("profile").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Training Profile</title><style>
body{font-family:system-ui,sans-serif;background:#101820;color:#eaf2f8;display:grid;place-items:center;min-height:100vh;margin:0}
main{background:#182733;border:1px solid #2f495b;border-radius:12px;padding:2rem;width:min(90vw,520px)}
code{display:block;background:#071117;padding:1rem;border-radius:7px;color:#75df95;word-break:break-all}a{color:#53c7e8}
</style></head><body><main><h1>Здравствуйте, student!</h1><p>Вы успешно прошли учебную проверку пароля.</p>
<p>Флаг:</p><code>{{.Flag}}</code><p><a href="/logout">Выйти</a></p></main></body></html>`))
)

type loginData struct{ Error string }
type profileData struct{ Flag string }

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", profileHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/logout", logoutHandler)

	server := &http.Server{Addr: ":8080", Handler: securityHeaders(mux)}
	log.Println("training service listens on :8080")
	log.Fatal(server.ListenAndServe())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderLogin(w, "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Намеренно отсутствуют rate limiting, блокировка и CAPTCHA.
	// Это делает сервис пригодным только для изолированного обучения.
	if r.FormValue("login") != trainingLogin || r.FormValue("password") != trainingPassword {
		renderLogin(w, "Неверный логин или пароль")
		return
	}

	token, err := newToken()
	if err != nil {
		http.Error(w, "cannot create session", http.StatusInternalServerError)
		return
	}
	sessionsMu.Lock()
	sessions[token] = true
	sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !hasSession(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = profilePage.Execute(w, profileData{Flag: trainingFlag})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func renderLogin(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	_ = loginPage.Execute(w, loginData{Error: message})
}

func hasSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	sessionsMu.RLock()
	valid := sessions[cookie.Value]
	sessionsMu.RUnlock()
	return valid
}

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
