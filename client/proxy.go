package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/binxio/gcloudconfig"
	"github.com/binxio/simple-iap-proxy/clusterinfo"
	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/elazarl/goproxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

// Proxy for GKE private master endpoints
type Proxy struct {
	cmd.RootCommand
	Audience              string
	ServiceAccount        string
	ConfigurationName     string
	UseDefaultCredentials bool
	TargetURL             string
	ProjectID             string
	ToGKEClusters         bool
	HostNames             []string
	HTTPProtocol          bool
	targetURL             *url.URL
	credentials           *google.Credentials
	tokenSource           oauth2.TokenSource
	certificate           *tls.Certificate
	clusterInfo           *clusterinfo.Cache
	hostNames             []*regexp.Regexp
}

// Run the proxy until stopped
func (p *Proxy) Run() error {
	var err error

	if p.UseDefaultCredentials && p.ConfigurationName != "" {
		return fmt.Errorf("specify either --use-default-credentials or --configuration, not both")
	}

	if !p.ToGKEClusters && len(p.HostNames) == 0 {
		return fmt.Errorf("at least --proxy-to or --proxy-to-gke must be specified")
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

	if p.ToGKEClusters {
		p.clusterInfo, err = clusterinfo.NewCache(ctx, p.ProjectID, p.credentials, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
	}

	p.hostNames = make([]*regexp.Regexp, 0, len(p.HostNames))
	for _, r := range p.HostNames {
		e, err := regexp.Compile(r)
		if err != nil {
			return fmt.Errorf("invalid proxy-to value, %s", err)
		}
		p.hostNames = append(p.hostNames, e)
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
		return fmt.Errorf("failed to create a token source for %s with audience %s, %s",
			p.ServiceAccount, p.Audience, err)
	}

	_, err = p.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to obtain token for %s, %s",
			p.ServiceAccount, err)
	}

	proxy := p.createProxy()

	srv := &http.Server{
		Handler:      proxy,
		Addr:         fmt.Sprintf(":%d", p.Port),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if p.HTTPProtocol {
		// I could not get the proxy on MacOS configured to connect using HTTPS :-(
		return srv.ListenAndServe()
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

// IsAllowedProxyEndpoint return true if the request is targets an allowed proxy endpoint
func (p *Proxy) IsAllowedProxyEndpoint() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		return p.clusterInfo != nil && p.clusterInfo.GetConnectInfoForEndpoint(req.URL.Host) != nil || goproxy.ReqHostMatches(p.hostNames...).HandleReq(req, ctx)
	}
}

// removeProxyHeaders before setting our own, copied from goproxy source as we need to set our own proxy-header
func removeProxyHeaders(ctx *goproxy.ProxyCtx, r *http.Request) {
	r.RequestURI = "" // this must be reset when serving a request with the client
	ctx.Logf("Sending request %v %v", r.Method, r.URL.String())
	// If no Accept-Encoding header exists, Transport will add the headers it can accept
	// and would wrap the response body with the relevant reader.
	r.Header.Del("Accept-Encoding")
	// curl can add that, see
	// https://jdebp.eu./FGA/web-proxy-connection-header.html
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	// Connection, Authenticate and Authorization are single hop Header:
	// http://www.w3.org/Protocols/rfc2616/rfc2616.txt
	// 14.10 Connection
	//   The Connection general-header field allows the sender to specify
	//   options that are desired for that particular connection and MUST NOT
	//   be communicated by proxies over further connections.

	// When server reads http request it sets req.Close to true if
	// "Connection" header contains "close".
	// https://github.com/golang/go/blob/master/src/net/http/request.go#L1080
	// Later, transfer.go adds "Connection: close" back when req.Close is true
	// https://github.com/golang/go/blob/master/src/net/http/transfer.go#L275
	// That's why tests that checks "Connection: close" removal fail
	if r.Header.Get("Connection") == "close" {
		r.Close = false
	}
	r.Header.Del("Connection")
}

// OnRequest inserts the IAP required token and renames an existing Authorization header
func (p *Proxy) OnRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	log.Printf("on request to %s", r.URL)

	token, err := p.tokenSource.Token()
	if err != nil {
		return r, goproxy.NewResponse(r,
			goproxy.ContentTypeText, http.StatusInternalServerError,
			fmt.Sprintf("failed to obtain IAP token, %s", err))
	}

	removeProxyHeaders(ctx, r)
	authorization := fmt.Sprintf("%s %s", token.Type(), token.AccessToken)
	r.Header.Set("Proxy-Authorization", authorization)
	RewriteRequestURL(r, p.targetURL)

	return r, nil
}

func (p *Proxy) createProxy() *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = p.Debug
	proxy.KeepHeader = true
	proxy.OnRequest(p.IsAllowedProxyEndpoint()).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(p.IsAllowedProxyEndpoint()).DoFunc(p.OnRequest)

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
