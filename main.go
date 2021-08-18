package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thinksystemio/package/response"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/*", NotFound)
	r.Get("/echo", Echo)

	// Loading
	r.Get("/loading", Loading)
	r.Get("/loading/*", Loading)

	// Cluster
	r.With(Middleware).Get("/{cluster}", Loading)
	r.With(Middleware).Get("/{cluster}/*", Loading)

	// Frontend
	r.With(Middleware).Get("/{cluster}/{app}", Frontend)
	r.With(Middleware).Get("/{cluster}/{app}/*", Frontend)

	// Backend
	r.With(Middleware).Get("/{cluster}/{app}/api", Backend)
	r.With(Middleware).Get("/{cluster}/{app}/api/*", Backend)
	r.With(Middleware).Post("/{cluster}/{app}/api", Backend)
	r.With(Middleware).Post("/{cluster}/{app}/api/*", Backend)

	http.ListenAndServe(":81", r)
}

// Echo allows pinging of this service
func Echo(w http.ResponseWriter, r *http.Request) {
	bg := GetBackground(r)
	bg.Res.SendDataWithStatusCode(w, "echo", http.StatusOK)
}

// NotFound redirects to the not found page
func NotFound(w http.ResponseWriter, r *http.Request) {
	bg := GetBackground(r)
	bg.Res.SendDataWithStatusCode(w, "not found", http.StatusNotFound)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bg := &Background{
			Cluster: chi.URLParam(r, "cluster"),
			App:     chi.URLParam(r, "app"),
			Res:     response.CreateResponse(),
		}

		if ok := VerifyCluster(bg); !ok {
			target := fmt.Sprintf("http://%s/%s", r.Host, bg.Cluster)
			http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			return
		}

		ctx := context.WithValue(r.Context(), BackgroundKey, bg)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Loading(w http.ResponseWriter, r *http.Request) {
	bg := GetBackground(r)

	target, err := LoadingAppURL(bg)
	target.Path = SkipNPathParams(r.URL.Path, 1)
	if err != nil {
		bg.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	ProxyRequest(w, r, target)
}

func Frontend(w http.ResponseWriter, r *http.Request) {
	bg := GetBackground(r)

	target, err := FrontendAppURL(bg)
	target.Path = SkipNPathParams(r.URL.Path, 2)
	if err != nil {
		bg.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	status, err := PingService(target)
	if status == http.StatusBadGateway && strings.Contains(r.URL.Path, "activate") {
		res, err := RequestWithTimeout(FrontendDeployURL(bg))
		if err != nil {
			bg.Res.AppendError(err)
		}
		bg.Res.SendWithStatusCode(w, res.StatusCode)
		return
	}
	if err != nil {
		bg.Res.SendErrorWithStatusCode(w, err, status)
		return
	}

	ProxyRequest(w, r, target)
}

func Backend(w http.ResponseWriter, r *http.Request) {
	bg := GetBackground(r)

	target, err := BackendAppURL(bg)
	target.Path = SkipNPathParams(r.URL.Path, 2)
	if err != nil {
		bg.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	status, err := PingService(target)
	if status == http.StatusBadGateway && strings.Contains(r.RequestURI, "activate") {
		res, err := RequestWithTimeout(BackendDeployURL(bg))
		if err != nil {
			bg.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
			return
		}
		bg.Res.SendWithStatusCode(w, res.StatusCode)
		return
	}
	if err != nil {
		bg.Res.SendErrorWithStatusCode(w, err, status)
		return
	}

	ProxyRequest(w, r, target)
}
