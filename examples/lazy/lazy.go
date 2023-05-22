package main

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/swdunlop/html-go/hog"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &log.Logger
	r := chi.NewRouter()
	r.Use(hog.Middleware())
	r.Get("/lazy", func(w http.ResponseWriter, r *http.Request) {
		hog.For(r).Info().Msg("taking a nap..")
		time.Sleep(1 * time.Second)
		http.Error(w, "I'm awake!", http.StatusOK)
	})
	http.ListenAndServe("localhost:8181", r)
}
