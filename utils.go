package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/thinksystemio/package-response"
)

type key int

const (
	BackgroundKey key = iota
)

var (
	registry = map[string]struct{}{}
)

type Background struct {
	Cluster string
	App     string
	Res     *response.Response
}

func GetBackground(r *http.Request) *Background {
	value, ok := r.Context().Value(BackgroundKey).(*Background)
	if !ok {
		return &Background{Res: response.CreateResponse()}
	}
	return value
}

func VerifyCluster(background *Background) bool {
	_, ok := registry[background.Cluster]
	if !ok {
		registry[background.Cluster] = struct{}{}
		http.Get(ClusterDeployURL(background))
	}
	return ok
}

func ClusterDeployURL(background *Background) string {
	return fmt.Sprintf(
		"http://service-backend-orchestration:81/%s",
		background.Cluster,
	)
}

func LoadingAppURL(background *Background) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://service-frontend-loading:80/%s",
		background.Cluster,
	)
	return url.Parse(URL)
}

func FrontendAppURL(background *Background) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://%s-service-frontend-%s:80",
		background.Cluster,
		background.App,
	)
	return url.Parse(URL)
}

func FrontendDeployURL(background *Background) string {
	return ThinksystemContainer(
		background.Cluster,
		"frontend",
		background.App,
		"80")
}

func BackendAppURL(background *Background) (*url.URL, error) {
	URL := fmt.Sprintf(
		"http://%s-service-backend-%s:81",
		background.Cluster,
		background.App,
	)
	return url.Parse(URL)
}

func BackendDeployURL(background *Background) string {
	return ThinksystemContainer(
		background.Cluster,
		"backend",
		background.App,
		"81")
}

func ThinksystemContainer(clusterName, appType, appName, appPort string) string {
	return fmt.Sprintf("http://service-backend-orchestration:81/%s/%s/%s/%s",
		clusterName,
		appType,
		appName,
		appPort,
	)
}

func RemoteContainer(clusterName, appType, appName, appPort, image, imageName string) string {
	return fmt.Sprintf("http://service-backend-orchestration:81/%s/%s/%s/%s/%s/%s",
		clusterName,
		appType,
		appName,
		appPort,
		image,
		imageName,
	)
}

func PingService(service *url.URL) (int, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/%s", service.Host, "echo"))
	if err != nil {
		return http.StatusBadGateway, err
	}
	return res.StatusCode, err
}

func ProxyRequest(w http.ResponseWriter, r *http.Request, target *url.URL) {
	reverseProxy := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.Header.Add("Cluster", strings.Split(target.Host, "-")[0])
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
			r.URL.Path = target.Path
		},
	}
	reverseProxy.ServeHTTP(w, r)
}

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
