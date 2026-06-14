package admin

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// router for /admin/*
func Router() http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(api chi.Router) {
		api.Post("/login", handleLogin)
		api.Post("/logout", handleLogout)
		api.Get("/me", handleMe)

		api.Group(func(p chi.Router) {
			p.Use(adminMiddleware)
			p.Post("/change-password", handleChangeAdminPassword)
			p.Get("/users", handleListUsers)
			p.Post("/users", handleCreateUser)
			p.Patch("/users/{id}", handleUpdateUser)
			p.Delete("/users/{id}", handleDeleteUser)
			p.Post("/users/{id}/apple-token", handleReplenish)
			p.Post("/users/{id}/apple-token/check", handleRecheck)
		})
	})

	r.Handle("/*", spaHandler())
	return r
}

func adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthed(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// deal with spaFS so we can go give actual files
func spaHandler() http.Handler {
	files := spaFS()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := chi.URLParam(r, "*")
		if p == "" {
			p = "index.html"
		}
		data, err := fs.ReadFile(files, p)
		if err != nil {
			// unknown path to SPA route, serve index.html
			data, err = fs.ReadFile(files, "index.html")
			if err != nil {
				http.Error(w, "admin UI not built / not in proper directory", http.StatusNotFound)
				return
			}
			p = "index.html"
		}
		http.ServeContent(w, r, p, time.Time{}, bytes.NewReader(data))
	})
}
