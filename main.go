package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	response "github.com/thinksystemio/package-response"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/*", NotFound)
	r.Get("/echo", Echo)

	// Loading
	r.Get("/loading", Loading)
	r.Get("/loading/*", Loading)

	// Instance
	r.With(Middleware).Get("/{instance}", Loading)
	r.With(Middleware).Get("/{instance}/*", Loading)

	// Frontend
	r.With(Middleware).Get("/{instance}/{app}", Frontend)
	r.With(Middleware).Get("/{instance}/{app}/*", Frontend)

	// Backend
	r.With(Middleware).Get("/{instance}/{app}/api", Backend)
	r.With(Middleware).Get("/{instance}/{app}/api/*", Backend)
	r.With(Middleware).Post("/{instance}/{app}/api", Backend)
	r.With(Middleware).Post("/{instance}/{app}/api/*", Backend)

	http.ListenAndServe(":80", r)
}

// Echo allows pinging of this service
func Echo(w http.ResponseWriter, r *http.Request) {
	appContext := GetAppContext(r)
	appContext.Res.SendDataWithStatusCode(w, "echo", http.StatusOK)
}

// NotFound redirects to the not found page
func NotFound(w http.ResponseWriter, r *http.Request) {
	appContext := GetAppContext(r)
	appContext.Res.SendDataWithStatusCode(w, "not found", http.StatusNotFound)
}

// Parse instance and app, place into context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appContext := &AppContext{
			Instance: chi.URLParam(r, "instance"),
			App:      chi.URLParam(r, "app"),
			Res:      response.CreateResponse(),
		}

		if ok := VerifyInstance(appContext); !ok {
			target := fmt.Sprintf("http://%s/%s", r.Host, appContext.Instance)
			http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			return
		}

		ctx := context.WithValue(r.Context(), AppContextKey, appContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Loading(w http.ResponseWriter, r *http.Request) {
	appContext := GetAppContext(r)

	target, err := LoadingAppURL(appContext)
	target.Path = SkipNPathParams(r.URL.Path, 1)
	if err != nil {
		appContext.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	ProxyRequest(w, r, target)
}

func Frontend(w http.ResponseWriter, r *http.Request) {
	appContext := GetAppContext(r)

	target, err := FrontendAppURL(appContext)
	target.Path = SkipNPathParams(r.URL.Path, 2)
	if err != nil {
		appContext.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	status, err := PingService(target)
	if status == http.StatusBadGateway && strings.Contains(r.URL.Path, "activate") {
		res, err := RequestWithTimeout(FrontendDeployURL(appContext))
		if err != nil {
			appContext.Res.AppendError(err)
		}
		appContext.Res.SendWithStatusCode(w, res.StatusCode)
		return
	}
	if err != nil {
		appContext.Res.SendErrorWithStatusCode(w, err, status)
		return
	}

	ProxyRequest(w, r, target)
}

func Backend(w http.ResponseWriter, r *http.Request) {
	appContext := GetAppContext(r)

	target, err := BackendAppURL(appContext)
	target.Path = SkipNPathParams(r.URL.Path, 2)
	if err != nil {
		appContext.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
		return
	}

	status, err := PingService(target)
	if status == http.StatusBadGateway && strings.Contains(r.RequestURI, "activate") {
		res, err := RequestWithTimeout(BackendDeployURL(appContext))
		if err != nil {
			appContext.Res.SendErrorWithStatusCode(w, err, http.StatusInternalServerError)
			return
		}
		appContext.Res.SendWithStatusCode(w, res.StatusCode)
		return
	}
	if err != nil {
		appContext.Res.SendErrorWithStatusCode(w, err, status)
		return
	}

	ProxyRequest(w, r, target)
}
