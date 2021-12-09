package gkeclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/binxio/gcloudconfig"
	"github.com/binxio/simple-iap-proxy/clusterinfo"
	"github.com/binxio/simple-iap-proxy/flags"
	"github.com/elazarl/goproxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Proxy for GKE private master endpoints
type Proxy struct {
	flags.RootCommand
	Audience              string
	ServiceAccount        string
	ConfigurationName     string
	UseDefaultCredentials bool
	TargetURL             string
	ProjectID             string
	targetURL             *url.URL
	credentials           *google.Credentials
	tokenSource           oauth2.TokenSource
	clusterInfo           *clusterinfo.Cache
	certificate           *tls.Certificate
}

// Run the proxy until stopped
func (p *Proxy) Run() error {
	var err error

	// mis-using the positional argument validator here.
	if p.UseDefaultCredentials && p.ConfigurationName != "" {
		return fmt.Errorf("specify either --use-default-credentials or --configuration, not both")
	}

	p.certificate, err = loadCertificate(p.KeyFile, p.CertificateFile)
	if err != nil {
		log.Fatal(err)
	}

	p.targetURL, err = url.Parse(p.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid target-url %s, %s", p.TargetURL, err)
	}

	if p.targetURL.Scheme != "https" {
		return fmt.Errorf("target-url must be https")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = p.getCredentials(ctx)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	p.clusterInfo, err = clusterinfo.NewCache(ctx, p.ProjectID, p.credentials, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	tokenConfig := impersonate.IDTokenConfig{
		TargetPrincipal: p.ServiceAccount,
		Audience:        p.Audience,
		IncludeEmail:    true,
	}

	p.tokenSource, err = impersonate.IDTokenSource(
		ctx,
		tokenConfig,
		option.WithTokenSource(p.credentials.TokenSource),
	)

	if err != nil {
		return fmt.Errorf("failed to create a token source for audience %s as %s, %s",
			p.Audience, p.ServiceAccount, err)
	}

	proxy := p.createProxy()

	srv := &http.Server{
		Handler:      proxy,
		Addr:         fmt.Sprintf(":%d", p.Port),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	return srv.ListenAndServeTLS(p.CertificateFile, p.KeyFile)
}

func (p *Proxy) getCredentials(ctx context.Context) error {
	var err error

	if p.UseDefaultCredentials || !gcloudconfig.IsGCloudOnPath() {
		p.credentials, err = google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform.read-only")
	} else {
		p.credentials, err = gcloudconfig.GetCredentials(p.ConfigurationName)
	}
	if err != nil {
		return fmt.Errorf("failed to obtain credentials, %s", err)
	}
	if p.ProjectID == "" {
		p.ProjectID = p.credentials.ProjectID
	}
	if p.ProjectID == "" {
		return fmt.Errorf("specify a --project as there is no default one")
	}
	return nil
}

// IsClusterEndpoint return true if the request is targeting an GKE cluster endpoint
func (p *Proxy) IsClusterEndpoint() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		return p.clusterInfo.GetConnectInfoForEndpoint(req.URL.Host) != nil
	}
}

// OnRequest inserts the IAP required token and renames an existing Authorization header
func (p *Proxy) OnRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	log.Printf("on request to %s", r.URL)

	token, err := p.tokenSource.Token()
	if err != nil {
		return r, goproxy.NewResponse(r,
			goproxy.ContentTypeText, http.StatusInternalServerError,
			fmt.Sprintf("Failed to obtained IAP token, %s", err))
	}

	// If there is a Authorization header, make it X-Real-Authorization header
	if authHeaders := r.Header.Values("Authorization"); len(authHeaders) > 0 {
		for _, v := range r.Header.Values("Authorization") {
			r.Header.Add("X-Real-Authorization", v)
		}
		r.Header.Del("Authorization")
	}

	authorization := fmt.Sprintf("%s %s", token.Type(), token.AccessToken)
	r.Header.Set("Authorization", authorization)
	RewriteRequestURL(r, p.targetURL)

	return r, nil
}

func (p *Proxy) createProxy() *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = p.Debug
	proxy.OnRequest(p.IsClusterEndpoint()).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(p.IsClusterEndpoint()).DoFunc(p.OnRequest)

	goproxy.GoproxyCa = *p.certificate
	tlsConfig := goproxy.TLSConfigFromCA(p.certificate)

	goproxy.OkConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectAccept,
		TLSConfig: tlsConfig,
	}
	goproxy.MitmConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectMitm,
		TLSConfig: tlsConfig,
	}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectHTTPMitm,
		TLSConfig: tlsConfig,
	}
	goproxy.RejectConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectReject,
		TLSConfig: tlsConfig,
	}

	return proxy
}
