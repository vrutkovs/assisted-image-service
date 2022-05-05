package servers

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type ServerInfo struct {
	HTTP            *http.Server
	HTTPS           *http.Server
	HTTPSKeyFile    string
	HTTPSCertFile   string
	HasBothHandlers bool
}

func New(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string, handler http.Handler) *ServerInfo {
	servers := ServerInfo{}
	if port := runHTTPSOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile); port != "" {
		servers.HTTPS = &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: handler,
		}
		servers.HTTPSCertFile = HTTPSCertFile
		servers.HTTPSKeyFile = HTTPSKeyFile
	}
	if port := runHTTPOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile); port != "" {
		servers.HTTP = &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: handler,
		}
	}
	return &servers
}

func runHTTPOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string) string {
	if (httpsPort == "" || HTTPSKeyFile == "" && HTTPSCertFile == "") && httpPort == "" {
		// Run HTTP listener on HTTPS port if httpPort is not set
		// This is default in podman deployment
		return httpsPort
	}
	if httpPort != "" {
		return httpPort
	}
	return ""
}

func runHTTPSOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string) string {
	if httpsPort != "" && HTTPSKeyFile != "" && HTTPSCertFile != "" {
		return httpsPort
	}
	return ""
}

func WillRunBothHandlers(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string) bool {
	httpHandlerPort := runHTTPOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile)
	httpsHandlerPort := runHTTPSOnPort(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile)
	return httpHandlerPort != "" && httpsHandlerPort != ""
}

func shutdown(name string, server *http.Server) {
	if err := server.Shutdown(context.TODO()); err != nil {
		log.Infof("%s shutdown failed: %v", name, err)
		if err := server.Close(); err != nil {
			log.Fatalf("%s emergency shutdown failed: %v", name, err)
		}
	} else {
		log.Infof("%s server terminated gracefully", name)
	}
}

func (s *ServerInfo) ListenAndServe() {
	if s.HTTP != nil {
		go s.httpListen()
	}
	if s.HTTPS != nil {
		go s.httpsListen()
	}
}

func (s *ServerInfo) Shutdown() bool {
	if s.HTTPS != nil {
		shutdown("HTTPS", s.HTTPS)
	}
	if s.HTTP != nil {
		shutdown("HTTP", s.HTTP)
	}
	return true
}

func (s *ServerInfo) httpListen() {
	log.Infof("Starting http handler on %s...", s.HTTP.Addr)
	if err := s.HTTP.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP listener closed: %v", err)
	}
}

func (s *ServerInfo) httpsListen() {
	log.Infof("Starting https handler on %s...", s.HTTPS.Addr)
	if err := s.HTTPS.ListenAndServeTLS(s.HTTPSCertFile, s.HTTPSKeyFile); err != http.ErrServerClosed {
		log.Fatalf("HTTPS listener closed: %v", err)
	}
}
