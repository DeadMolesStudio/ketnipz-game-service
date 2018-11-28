package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"
)

type key int

const (
	KeyIsAuthenticated key = iota
	KeySessionID
	KeyUserID
)

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", "https://dmstudio.now.sh")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, User-Agent, Cache-Control, Accept, X-Requested-With, If-Modified-Since, Origin")

		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SessionMiddleware(next http.HandlerFunc, sm *session.SessionManager) http.HandlerFunc {
	// middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c, err := r.Cookie("session_id")
		if err == nil {
			uid, err := sm.Get(c.Value)
			switch err {
			case nil:
				ctx = context.WithValue(ctx, KeyIsAuthenticated, true)
				ctx = context.WithValue(ctx, KeySessionID, c.Value)
				ctx = context.WithValue(ctx, KeyUserID, uid)
			case session.ErrKeyNotFound:
				// delete unvalid cookie
				c.Expires = time.Now().AddDate(0, 0, -1)
				http.SetCookie(w, c)
				ctx = context.WithValue(ctx, KeyIsAuthenticated, false)
			default:
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else { // ErrNoCookie
			ctx = context.WithValue(ctx, KeyIsAuthenticated, false)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("[PANIC]: ", err, " at ", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func AccessLogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		logger.Infow(r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"work_time", time.Since(start).String(),
		)
	})
}
