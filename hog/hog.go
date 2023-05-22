// Package hog provides middleware for logging HTTP requests in a service and recovering from panics using
// zerolog and a minimum of spam.
package hog

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/rs/zerolog"
)

// From will return the logger from the provided context.  This is introduced into a request
// context by the Middleware.  We use the same context key as zerolog Ctx and WithContext to
// improve interoperability.
func From(ctx context.Context, injects ...func(zerolog.Context) zerolog.Context) *zerolog.Logger {
	log := zerolog.Ctx(ctx)

	if len(injects) == 0 || log == nil {
		return log
	}
	z := log.With()
	for _, inject := range injects {
		z = inject(z)
	}
	next := z.Logger()
	return &next
}

// Middleware returns a middleware that logs requests and recovers from panics.  The inject functions (if present) can
// extend the request log context with information, such as the user or request ID.  See the For function for a list
// of fields added to the log context.  Fields logged after the middleware completes:
//
//   - status: the HTTP status code of the response
//   - wrote: the number of bytes written to the response
//   - took: the number of milliseconds the request took to process
//   - panic: the panic message, if the request panicked
//   - stack: the stack trace, if the request panicked, as a list of strings where each string is a function and line.
func Middleware(injects ...func(zerolog.Context) zerolog.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			log := For(r, injects...)
			r = r.WithContext(log.WithContext(r.Context()))
			defer logResponse(log, ww, r, start)
			next.ServeHTTP(ww, r)
		})
	}
}

// With returns a new context with the provided injectors applied to the log context.  If there are no injectors, then
// the context is returned unchanged.
func With(ctx context.Context, injects ...func(zerolog.Context) zerolog.Context) context.Context {
	if len(injects) == 0 {
		return ctx
	}
	log := zerolog.Ctx(ctx)
	z := log.With()
	for _, inject := range injects {
		z = inject(z)
	}
	return z.Logger().WithContext(ctx)
}

// For will return a logger for the provided request, applying any injectors to the log context.  It will add the
// following fields to the log context:
//
//   - remote_addr: the remote address of the request
//   - method: the HTTP method of the request
//   - path: the path of the request
//
// NOTE: This is not necessary if you are using Middleware.
func For(r *http.Request, injects ...func(zerolog.Context) zerolog.Context) *zerolog.Logger {
	ctx := r.Context()
	prev := zerolog.Ctx(ctx)
	z := prev.With().
		Str(`remote_addr`, r.RemoteAddr).
		Str(`method`, r.Method).
		Str(`path`, r.URL.Path)
	for _, inject := range injects {
		z = inject(z)
	}
	log := z.Logger()
	return &log
}

func logResponse(log *zerolog.Logger, ww middleware.WrapResponseWriter, r *http.Request, start time.Time) {
	var evt *zerolog.Event
	if e := recover(); e != nil {
		if e == http.ErrAbortHandler {
			panic(e) // rethrow, http will handle it.
		}
		evt = logRecovery(log, e)
	} else {
		status := ww.Status()
		if status >= 500 {
			evt = log.Error()
		} else if status >= 400 {
			evt = log.Warn()
		} else {
			evt = log.Info()
		}
		evt = evt.Int(`status`, status).
			Int(`wrote`, ww.BytesWritten()).
			Int64(`took`, time.Since(start).Milliseconds())
	}
	evt.Msg(``)
}

func logRecovery(log *zerolog.Logger, e any) *zerolog.Event {
	evt := log.WithLevel(zerolog.PanicLevel)
	evt = addStackTrace(evt, 4)
	evt = evt.Str(`panic`, fmt.Sprint(e))
	return evt
}

func addStackTrace(evt *zerolog.Event, skip int) *zerolog.Event {
	var calls [64]uintptr
	n := runtime.Callers(skip+1, calls[:])
	stack := make([]string, 0, n)
	for _, pc := range calls[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		_, line := fn.FileLine(pc)
		stack = append(stack, fmt.Sprintf(`%v:%v`, fn.Name(), line))
	}
	return evt.Strs(`stack`, stack)
}

// NOTE(swdunlop): We are not concerned with github.com/pkg/errors stack tracing here because
// we are recovering from a panic, and therefore the stack trace is visible and more detailed
// than that produced by pkg/errors.Wrap.
