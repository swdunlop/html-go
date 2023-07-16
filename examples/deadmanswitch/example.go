package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/swdunlop/html-go"
	"github.com/swdunlop/html-go/deadmanswitch"
	"github.com/swdunlop/html-go/hog"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &log.Logger
	r := chi.NewRouter()
	r.Use(hog.Middleware())
	start := time.Now().Format(`2006-01-02 15:04:05`)
	dms := deadmanswitch.New(deadmanswitch.ReloadOnReconnect())
	r.Method(`GET`, dms.Path(), dms)
	r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
		render(w, r, 200,
			html.HTML(`<html><body>`),
			html.HTML(`<h1>Server Started: `),
			html.Text(start),
			html.HTML(`</h1>`),
			dms,
			html.HTML(`</body></html>`),
		)
	})
	http.ListenAndServe("localhost:8181", r)
}

func render(w http.ResponseWriter, r *http.Request, status int, content ...html.Content) {
	p := html.Append(make([]byte, 0, 16384), content...)
	h := w.Header()
	h.Set(`Content-Type`, `text/html; encoding=utf-8`)
	h.Set(`Content-Length`, strconv.Itoa(len(p)))
	w.WriteHeader(status)
	_, err := w.Write(p)
	if err != nil {
		hog.For(r).Warn().Err(err).Msg(``)
	}
}
