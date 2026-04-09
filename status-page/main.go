package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	// порт, на котором запускается сервер
	listenAddr = ":8080"
)

//
// структуры данных,
// которые передаются в HTML шаблоны
//

// данные для страницы с сообщением
type PageData struct {
	Content template.HTML
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
	// создаем HTTP router
	//
	mux := http.NewServeMux()
	// раздаем статику
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// главная страница
	mux.HandleFunc("/", handleIndex)
	// админка
	mux.HandleFunc(adminPath, basicAuth(handleAdmin))

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
		Content: htmlContent,
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
