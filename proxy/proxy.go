package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"gopenbridge/config"
)

// NewReverseProxy creates a reverse proxy pointing at cfg.BaseURL.
func NewReverseProxy(cfg *config.Config) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		// Set upstream URL to combined base URL and incoming path
		// Forward original path and query parameters
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		// Set Authorization header
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	return proxy, nil
}
