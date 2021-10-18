package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	response "github.com/thinksystemio/package-response"
)

//
// HTTP Helpers
//

// ProxyRequest proxies a request to a given instance.
func ProxyRequest(w http.ResponseWriter, r *http.Request, target *url.URL) {
	reverseProxy := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.Header.Add("Instance", strings.Split(target.Host, "-")[0])
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
			r.URL.Path = target.Path
		},
	}
	reverseProxy.ServeHTTP(w, r)
}

// RequestWithTimeout sends a request with a given context that
// will timeout.
func RequestWithTimeout(url string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// PingService checks is a service has been deployed
// for a given instance.
func PingService(service *url.URL) (int, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/%s", service.Host, "echo"))
	if err != nil {
		return http.StatusBadGateway, err
	}
	return res.StatusCode, err
}

// SkipNPathParams trims the first n items in a path. For examples,
// if the path is test/route/path with n=1, it will return
// route/path.
func SkipNPathParams(path string, n int) string {
	if n < 0 {
		return "/"
	}

	slashCount := 0
	for i, c := range path {
		if string(c) == "/" {
			slashCount += 1
		}
		if slashCount == n+1 {
			return path[i:]
		}
	}

	return "/"
}

//
// Deployment Helpers
//

var (
	registry = map[string]struct{}{}
)

type key int

const (
	AppContextKey key = iota
)

type AppContext struct {
	Instance string
	App      string
	Res      *response.Response
}

// GetAppContext gets the current context being passed
// through the application
func GetAppContext(r *http.Request) *AppContext {
	value, ok := r.Context().Value(AppContextKey).(*AppContext)
	if !ok {
		return &AppContext{Res: response.CreateResponse()}
	}
	return value
}

// VerifyInstance ensures that the registry has an instance
// already created. If the instance is not already created,
// it will create one with all the deployments.
func VerifyInstance(appContext *AppContext) bool {
	_, ok := registry[appContext.Instance]
	fmt.Println("verify")
	fmt.Println(ok)
	return ok
}

// DeployInstance sends a request to the orchestration
// server and tells it to create a new instance with
// all of the necessary containers.
func DeployInstance(appContext *AppContext) bool {
	url := fmt.Sprintf(
		"http://backend-orchestration:80/api/%s",
		appContext.Instance,
	)

	res, _ := http.Get(url)

	fmt.Println("add to registry")
	fmt.Println(appContext.Instance)
	if res.StatusCode == http.StatusOK {
		registry[appContext.Instance] = struct{}{}
		fmt.Println("added to registry" + appContext.Instance)
		return true
	}

	return false
}

func LoadingAppURL(appContext *AppContext) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://frontend-loading:80/%s",
		appContext.Instance,
	)
	return url.Parse(URL)
}

func FrontendAppURL(appContext *AppContext) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://%s-frontend-%s:80",
		appContext.Instance,
		appContext.App,
	)
	return url.Parse(URL)
}

func FrontendDeployURL(appContext *AppContext) string {
	URL := fmt.Sprintf(
		"http://backend-orchestration:80/api/%s/frontend-%s",
		appContext.Instance,
		appContext.App,
	)
	return URL
}

func BackendAppURL(appContext *AppContext) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://%s-backend-%s:80",
		appContext.Instance,
		appContext.App,
	)
	return url.Parse(URL)
}

func BackendDeployURL(appContext *AppContext) string {
	URL := fmt.Sprintf(
		"http://backend-orchestration:80/api/%s/backend-%s",
		appContext.Instance,
		appContext.App,
	)
	return URL
}
