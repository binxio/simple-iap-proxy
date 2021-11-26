//   Copyright 2021 binx.io B.V.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/impersonate"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
)

type ProxyHandler struct {
	proxy            *httputil.ReverseProxy
	target           *url.URL
	tokenSource      oauth2.TokenSource
	debug            bool
	renameAuthHeader bool
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if h.renameAuthHeader {
		// If there is a X-Real-Authorization header, make it Authorization header
		if realAuthHeaders := r.Header.Values("X-Real-Authorization"); len(realAuthHeaders) > 0 {
			for _, v := range r.Header.Values("X-Real-Authorization") {
				r.Header.Add("Authorization", v)
			}
		} else {
			// If there is a Authorization header, make it a X-Real-Authorization header
			for _, v := range r.Header.Values("Authorization") {
				r.Header.Add("X-Real-Authorization", v)
			}
			r.Header.Del("Authorization")
		}
	}

	if h.tokenSource != nil {
		if token, err := h.tokenSource.Token(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to obtained IAP token, %s", err), http.StatusInternalServerError)
			return
		} else {
			authorization := fmt.Sprintf("%s %s", token.Type(), token.AccessToken)
			if r.Header.Get("Authorization") == "" {
				r.Header.Set("Authorization", authorization)
			} else {
				r.Header.Set("Proxy-Authorization", authorization)
			}
		}
	}
	r.Host = h.target.Host

	if h.debug {
		x, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("failed to dump the response body, %s", err)
		} else {
			log.Println(fmt.Sprintf("%q", x))
		}
	}

	rec := httptest.NewRecorder()
	h.proxy.ServeHTTP(rec, r)

	if h.debug {
		x, err := httputil.DumpResponse(rec.Result(), true)
		if err != nil {
			log.Printf("failed to dump the response body, %s", err)
		} else {
			log.Println(fmt.Sprintf("%q", x))
		}
	}

	for key, values := range rec.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(rec.Code)
	_, err := rec.Body.WriteTo(w)
	if err != nil {
		log.Printf("error writing body, %s", err)
	}
}

func main() {
	var insecure bool
	var debug bool
	var renameAuthHeader bool
	var targetURL string
	var listenPort string
	var tokenSource oauth2.TokenSource
	var certificateFile string
	var keyFile string
	var audience string
	var serviceAccount string

	flag.StringVar(&targetURL, "target-url", "", "to forward HTTP requests to")
	flag.StringVar(&serviceAccount, "service-account", "", "to impersonate")
	flag.StringVar(&audience, "iap-audience", "", "to call a service behind the Identity Aware Proxy")
	flag.StringVar(&certificateFile, "certificate-file", "", "for TLS")
	flag.StringVar(&keyFile, "key-file", "", "for TLS")
	flag.BoolVar(&insecure, "insecure", false, "allows insecure TLS connections")
	flag.BoolVar(&renameAuthHeader, "rename-auth-header", false, "rename Authorization Header to X-Real-Authorization to workaround IAP limitation")
	flag.BoolVar(&debug, "debug", false, "logs requests and responses")
	flag.Parse()
	if targetURL == "" {
		log.Fatal("option -target-url is missing")
	}

	if certificateFile != "" && keyFile == "" || keyFile != "" && certificateFile == "" {
		log.Fatalf("both -certificate-file and -certificate-key are required.")
	} else if keyFile != "" {
		if s, err := os.Stat(keyFile); err != nil {
			log.Fatalf("invalid option -key-file, %s", err)
		} else {
			if s.IsDir() {
				log.Fatalf("option -key-file must be a file")
			}
		}
		if s, err := os.Stat(certificateFile); err != nil {
			log.Fatalf("invalid option -certificate-file, %s", err)
		} else {
			if s.IsDir() {
				log.Fatalf("option -certificate-file must be a file")
			}
		}
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("failed to parse target URL %s, %s", targetURL, err)
	}
	if target.Scheme != "https" {
		log.Fatalf("invalid target url %s, only HTTPS target urls are supported", targetURL)
	}

	listenPort = os.Getenv("PORT")
	if listenPort == "" {
		if keyFile == "" {
			listenPort = "8080"
		} else {
			listenPort = "8443"
		}
	}

	if port, err := strconv.ParseUint(listenPort, 10, 64); err != nil || port > 65535 {
		log.Fatalf("the environment variable PORT is not a valid port number")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	if audience != "" {
		if serviceAccount != "" {
			tokenSource, err = impersonate.IDTokenSource(context.Background(), impersonate.IDTokenConfig{
				TargetPrincipal: serviceAccount,
				Audience:        audience,
				IncludeEmail:    true,
			})
			if err != nil {
				log.Fatalf("failed to create a token source for audience %s as %s, %s",
					audience, serviceAccount, err)
			}
		} else {
			tokenSource, err = idtoken.NewTokenSource(context.Background(), audience)
			if err != nil {
				log.Fatalf("failed to create a token source for audience %s, %s",
					audience, err)
			}
		}
	}

	if tokenSource != nil {
		if _, err := tokenSource.Token(); err != nil {
			if serviceAccount != "" {
				log.Fatalf("cannot create id token for audience %s as %s, %s", audience, serviceAccount, err)
			} else {
				log.Fatalf("cannot create id token for audience %s, %s", audience, err)
			}
		}
	}

	if insecure {
		proxy.Transport =
			&http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
	}

	http.Handle("/", &ProxyHandler{
		proxy:            proxy,
		target:           target,
		tokenSource:      tokenSource,
		renameAuthHeader: renameAuthHeader,
		debug:            debug})

	if keyFile == "" {
		err = http.ListenAndServe(":"+listenPort, nil)
	} else {
		err = http.ListenAndServeTLS(":"+listenPort, certificateFile, keyFile, nil)
	}

	if err != nil {
		log.Fatalf("server failed, %s", err)
	}

}
