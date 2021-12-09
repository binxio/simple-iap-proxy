package gkeserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/binxio/simple-iap-proxy/clusterinfo"
	"github.com/binxio/simple-iap-proxy/flags"
	"golang.org/x/oauth2/google"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// ReverseProxy provides the runtime configuration of the Reverse Proxy
type ReverseProxy struct {
	flags.RootCommand
	clusterInfo *clusterinfo.Cache
}

func (p *ReverseProxy) retrieveClusterInfo(ctx context.Context) error {
	credentials, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform.read-only")
	if err != nil {
		return err
	}
	if p.ProjectID == "" {
		p.ProjectID = credentials.ProjectID
	}
	if p.ProjectID == "" {
		return fmt.Errorf("specify a --project as there is no default one")
	}

	p.clusterInfo, err = clusterinfo.NewCache(ctx, p.ProjectID, credentials, 5*time.Minute)
	return err
}
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "service is healthy\n")
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	clusterInfo := p.clusterInfo.GetConnectInfoForEndpoint(r.Host)
	if clusterInfo == nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("%s is not a cluster endpoint", r.Host)))
		return
	}

	targetURL, err := url.Parse(fmt.Sprintf("https://%s", r.Host))
	if clusterInfo == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to parse URL https://%s, %s", r.Host, err)))
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: clusterInfo.RootCAs,
		},
	}

	// If there is a X-Real-Authorization header, make it Authorization header
	if realAuthHeaders := r.Header.Values("X-Real-Authorization"); len(realAuthHeaders) > 0 {
		r.Header.Del("Authorization")
		for _, v := range r.Header.Values("X-Real-Authorization") {
			r.Header.Add("Authorization", v)
		}
	}

	proxy.ServeHTTP(w, r)
}

// Run the reverse proxy until stopped
func (p *ReverseProxy) Run() error {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = p.retrieveClusterInfo(ctx); err != nil {
		return fmt.Errorf("failed to retrieve cluster information, %s", err)
	}

	http.Handle("/", p)
	http.HandleFunc("/__health", healthCheckHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", p.Port),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	err = srv.ListenAndServeTLS(p.CertificateFile, p.KeyFile)
	return err
}
