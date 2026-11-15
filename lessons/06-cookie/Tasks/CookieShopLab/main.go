package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const (
	initialBalance = 10_000
	flagPrice      = 1_000_000
)

type user struct {
	Login    string
	Password string
}

type product struct {
	ID          string
	Name        string
	Description string
	Price       int
	Flag        string
}

type pageData struct {
	Title     string
	Login     string
	Balance   int
	Products  []product
	Error     string
	Success   string
	Purchased *product
}

type app struct {
	templates *template.Template
	mu        sync.RWMutex
	users     map[string]user
	products  []product
}

func main() {
	a := &app{
		templates: template.Must(template.ParseGlob("templates/*.html")),
		users:     make(map[string]user),
		products: []product{
			{ID: "sticker", Name: "CTF-стикерпак", Description: "Набор наклеек для ноутбука.", Price: 50},
			{ID: "guide", Name: "Памятка участника", Description: "Краткий чек-лист перед соревнованием.", Price: 250},
			{ID: "hoodie", Name: "Худи факультатива", Description: "Лимитированный мерч команды.", Price: 3_500},
			{ID: "flag", Name: "Секретный флаг", Description: "Эксклюзивный цифровой товар для VIP-покупателей.", Price: flagPrice, Flag: "CTF{cookies_should_not_control_money}"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /", a.store)
	mux.HandleFunc("GET /login", a.loginForm)
	mux.HandleFunc("POST /login", a.login)
	mux.HandleFunc("GET /signup", a.signupForm)
	mux.HandleFunc("POST /signup", a.signup)
	mux.HandleFunc("POST /buy", a.buy)
	mux.HandleFunc("POST /logout", a.logout)

	server := &http.Server{
		Addr:    ":8080",
		Handler: noStore(mux),
	}

	log.Println("Cookie Shop Lab listening on :8080")
	log.Fatal(server.ListenAndServe())
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (a *app) store(w http.ResponseWriter, r *http.Request) {
	login, balance, ok := currentIdentity(r)
	if !ok || !a.userExists(login) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	a.render(w, "store.html", pageData{
		Title:    "Cookie Shop",
		Login:    login,
		Balance:  balance,
		Products: a.products,
	})
}

func (a *app) loginForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, "login.html", pageData{Title: "Вход — Cookie Shop"})
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")

	a.mu.RLock()
	u, found := a.users[login]
	a.mu.RUnlock()
	if !found || u.Password != password {
		a.render(w, "login.html", pageData{Title: "Вход — Cookie Shop", Error: "Неверный логин или пароль."})
		return
	}

	setIdentityCookies(w, login, initialBalance)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) signupForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, "signup.html", pageData{Title: "Регистрация — Cookie Shop"})
}

func (a *app) signup(w http.ResponseWriter, r *http.Request) {
	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")
	if len(login) < 3 || len(password) < 4 {
		a.render(w, "signup.html", pageData{Title: "Регистрация — Cookie Shop", Error: "Логин — от 3 символов, пароль — от 4 символов."})
		return
	}

	a.mu.Lock()
	_, exists := a.users[login]
	if !exists {
		a.users[login] = user{Login: login, Password: password}
	}
	a.mu.Unlock()
	if exists {
		a.render(w, "signup.html", pageData{Title: "Регистрация — Cookie Shop", Error: "Такой логин уже занят."})
		return
	}

	setIdentityCookies(w, login, initialBalance)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) buy(w http.ResponseWriter, r *http.Request) {
	login, balance, ok := currentIdentity(r)
	if !ok || !a.userExists(login) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	p, found := a.productByID(r.FormValue("product"))
	if !found {
		http.NotFound(w, r)
		return
	}
	if balance < p.Price {
		a.render(w, "store.html", pageData{
			Title: "Cookie Shop", Login: login, Balance: balance, Products: a.products,
			Error: "Недостаточно средств для покупки этого товара.",
		})
		return
	}

	newBalance := balance - p.Price
	setIdentityCookies(w, login, newBalance)
	a.render(w, "store.html", pageData{
		Title: "Cookie Shop", Login: login, Balance: newBalance, Products: a.products,
		Success: "Оплата прошла успешно.", Purchased: &p,
	})
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{"login", "balance"} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *app) userExists(login string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, exists := a.users[login]
	return exists
}

func (a *app) productByID(id string) (product, bool) {
	for _, p := range a.products {
		if p.ID == id {
			return p, true
		}
	}
	return product{}, false
}

func (a *app) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "Ошибка отображения страницы", http.StatusInternalServerError)
	}
}

// currentIdentity намеренно доверяет значениям cookie. Это CTF-уязвимость:
// баланс и логин не сверяются с серверным хранилищем и не имеют подписи.
func currentIdentity(r *http.Request) (string, int, bool) {
	loginCookie, loginErr := r.Cookie("login")
	balanceCookie, balanceErr := r.Cookie("balance")
	if loginErr != nil || balanceErr != nil || loginCookie.Value == "" {
		return "", 0, false
	}

	balance, err := strconv.Atoi(balanceCookie.Value)
	if err != nil || balance < 0 {
		return "", 0, false
	}
	return loginCookie.Value, balance, true
}

func setIdentityCookies(w http.ResponseWriter, login string, balance int) {
	// ВАЖНО: cookie намеренно не подписаны в рамках лабораторной.
	common := http.Cookie{Path: "/", SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, &http.Cookie{Name: "login", Value: login, Path: common.Path, SameSite: common.SameSite})
	http.SetCookie(w, &http.Cookie{Name: "balance", Value: fmt.Sprintf("%d", balance), Path: common.Path, SameSite: common.SameSite})
}
