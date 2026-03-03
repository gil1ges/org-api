package app

import (
	"log"
	"net/http"
	"time"

	"org-api/internal/config"
	"org-api/internal/handlers"
	"org-api/internal/repository"
	"org-api/internal/service"

	"gorm.io/gorm"
)

func NewServer(cfg config.Config, db *gorm.DB) *http.Server {
	repo := repository.NewGormRepository(db)
	svc := service.NewOrgService(repo)
	handler := handlers.New(svc)

	mux := http.NewServeMux()
	handler.Register(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:         cfg.Addr(),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(start))
	})
}
