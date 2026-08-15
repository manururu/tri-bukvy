package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yuin/goldmark"
)

// Пути к файлам и конфиги
var (
	// директория, где хранится markdown
	dataDir = "data"
	// файл с сообщением
	messageMD = filepath.Join(dataDir, "message.md")
	// парсим HTML шаблоны из папки templates
	templates = template.Must(template.ParseFiles("templates/index.html", "templates/admin.html"))
	// путь к админке
	// например:
	// ADMIN_PATH=/admin
	adminPath = strings.TrimSpace(os.Getenv("ADMIN_PATH"))
	// логин и пароль для basic auth
	adminUser = os.Getenv("ADMIN_USERNAME")
	adminPass = os.Getenv("ADMIN_PASSWORD")
	// адрес хоста для проверки доступности (host:port),
	// например: PING_TARGET=192.168.1.10:443
	// если не задан — плашка не показывается, /api/status не регистрируется
	pingTarget = strings.TrimSpace(os.Getenv("PING_TARGET"))
	// последний результат проверки (nil, пока проверка не выполнена)
	pingState atomic.Pointer[PingResult]
	// порт, на котором запускается сервер
	listenAddr = ":8080"
)

//
// структуры данных,
// которые передаются в HTML шаблоны
//

// данные для страницы с сообщением
type PageData struct {
	Content     template.HTML
	PingEnabled bool   // показывать ли плашку доступности
	PingStatus  string // "ok", "fail" или "pending"
}

// результат проверки доступности хоста
type PingResult struct {
	OK        bool      // доступен ли хост
	CheckedAt time.Time // время последней проверки
}

// данные для админки
type AdminData struct {
	Content string
}

func main() {
	//
	// проверка обязательных настроек
	//
	if adminPath == "" {
		log.Fatal("ADMIN_PATH is required, e.g.: /abcdef012345-panel")
	}
	if !strings.HasPrefix(adminPath, "/") {
		adminPath = "/" + adminPath
	}
	if adminUser == "" || adminPass == "" {
		log.Fatal("ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}

	//
	// создаем файл message.md,
	// если он еще не существует
	//
	ensureData()

	//
	// проверка доступности хоста
	//
	if pingTarget != "" {
		// цель должна быть в формате host:port
		if _, _, err := net.SplitHostPort(pingTarget); err != nil {
			log.Fatalf("PING_TARGET must be host:port, got %q: %v", pingTarget, err)
		}
		// первая проверка синхронная — статус известен ещё до старта сервера
		checkPing()
		// дальше проверяем каждые 5 минут в фоне
		go pingLoop()
	}

	//
	// создаем HTTP router
	//
	mux := http.NewServeMux()
	// раздаем статику
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// главная страница
	mux.HandleFunc("/", handleIndex)
	// админка
	mux.HandleFunc(adminPath, basicAuth(handleAdmin))
	// API: текущее состояние проверки (публичное, без авторизации)
	if pingTarget != "" {
		mux.HandleFunc("/api/status", handleStatus)
	}

	log.Printf("listening on %s", listenAddr)
	log.Printf("admin path configured")
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

// создает файл message.md,
// если его еще нет
func ensureData() {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	// если файла нет — создаем дефолтный текст
	if _, err := os.Stat(messageMD); os.IsNotExist(err) {
		if err := os.WriteFile(messageMD, []byte("# hello\n"), 0644); err != nil {
			log.Fatal(err)
		}
	}
}

// обработчик главной страницы
func handleIndex(w http.ResponseWriter, r *http.Request) {
	// разрешаем только /
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// читаем markdown файл
	md, err := os.ReadFile(messageMD)
	if err != nil {
		http.Error(w, "failed to read message", http.StatusInternalServerError)
		return
	}

	// преобразуем markdown -> html
	htmlContent, err := renderMarkdown(md)
	if err != nil {
		http.Error(w, "failed to render markdown", http.StatusInternalServerError)
		return
	}

	// вставляем HTML в шаблон index.html
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", PageData{
		Content:     htmlContent,
		PingEnabled: pingTarget != "",
		PingStatus:  pingStatus(),
	}); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

// обработчик админки
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// открываем страницу редактирования
	case http.MethodGet:
		md, err := os.ReadFile(messageMD)
		if err != nil {
			http.Error(w, "failed to read message", http.StatusInternalServerError)
			return
		}
		// вставляем markdown в textarea
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.ExecuteTemplate(w, "admin.html", AdminData{
			Content: string(md),
		}); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
	// сохраняем изменения
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		// получаем текст из textarea
		content := r.FormValue("content")
		// сохраняем markdown в файл
		if err := os.WriteFile(messageMD, []byte(content), 0644); err != nil {
			http.Error(w, "failed to save message", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// преобразует markdown в html
func renderMarkdown(md []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert(md, &buf); err != nil {
		return "", err
	}
	// помечаем HTML как безопасный,
	// чтобы template не экранировал его
	return template.HTML(buf.String()), nil
}

// возвращает текущий статус: "ok", "fail" или "pending"
func pingStatus() string {
	if v := pingState.Load(); v != nil {
		if v.OK {
			return "ok"
		}
		return "fail"
	}
	return "pending"
}

// одна проверка доступности PING_TARGET по TCP
func checkPing() {
	conn, err := net.DialTimeout("tcp", pingTarget, 3*time.Second)
	if err != nil {
		pingState.Store(&PingResult{OK: false, CheckedAt: time.Now()})
		log.Printf("ping %s: unreachable: %v", pingTarget, err)
		return
	}
	conn.Close()
	pingState.Store(&PingResult{OK: true, CheckedAt: time.Now()})
	log.Printf("ping %s: reachable", pingTarget)
}

// фоновая проверка каждые 5 минут
// (интервал должен совпадать с setInterval в templates/index.html)
func pingLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		checkPing()
	}
}

// обработчик /api/status: отдаёт сохранённое состояние, сам проверку не делает
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if s := pingStatus(); s == "pending" {
		fmt.Fprint(w, `{"ok": null}`)
	} else {
		fmt.Fprintf(w, `{"ok": %t}`, s == "ok")
	}
}

// basic auth middleware
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != adminUser || pass != adminPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
