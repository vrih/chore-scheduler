package server

import (
	"crypto/md5"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(
	template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html"),
)

var templateFuncs = template.FuncMap{
	"cssVersion":  func() string { return cssVersion },
	"effortLabel": effortLabel,
	"effortDots":  effortDots,
	"floorLabel":  floorLabel,
	"add":         func(a, b int) int { return a + b },
	"seq": func(lo, hi int) []int {
		out := make([]int, 0, hi-lo+1)
		for i := lo; i <= hi; i++ {
			out = append(out, i)
		}
		return out
	},
}

func floorLabel(n int) string {
	switch n {
	case 0:
		return "Ground floor"
	case 1:
		return "First floor"
	case 2:
		return "Second floor"
	case 3:
		return "Loft"
	default:
		return fmt.Sprintf("Floor %d", n)
	}
}

func effortDots(effort int) template.HTML {
	out := `<span class="effort-dots">`
	for i := 1; i <= 3; i++ {
		if i <= effort {
			out += `<span class="dot dot-on"></span>`
		} else {
			out += `<span class="dot dot-off"></span>`
		}
	}
	out += `</span>`
	return template.HTML(out)
}

func effortLabel(effort int) string {
	switch effort {
	case 1:
		return "Quick"
	case 2:
		return "Medium"
	case 3:
		return "Long"
	default:
		return "?"
	}
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

//go:embed static
var staticFS embed.FS

var cssVersion string

func init() {
	data, _ := staticFS.ReadFile("static/app.css")
	h := md5.Sum(data)
	cssVersion = fmt.Sprintf("%x", h[:4])
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	sub, _ := fs.Sub(staticFS, "static")
	http.StripPrefix("/static/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
}

func (s *Server) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Service-Worker-Allowed", "/")
	data, _ := staticFS.ReadFile("static/sw.js")
	w.Write(data)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	data, _ := staticFS.ReadFile("static/manifest.json")
	w.Write(data)
}
