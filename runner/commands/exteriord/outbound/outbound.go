package outbound

import (
	"crypto/subtle"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	addr      string
	authKey   string
	server    *http.Server
	msgRouter *msgrouter.MsgRouter
}

func NewServer(addr string, authKey string, msgRouter *msgrouter.MsgRouter) *Server {
	return &Server{
		addr:      addr,
		authKey:   authKey,
		msgRouter: msgRouter,
	}
}

func (self *Server) HandleMonitor(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Auth-Key")), []byte(self.authKey)) == 0 {
		log.Error("Connection is closed because it has no valid auth key")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	client := self.msgRouter.Subscribe(msgrouter.NotifyLatest("serverStatus"))
	defer self.msgRouter.Unsubscribe(client)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastSent := time.Now().Add(time.Hour)

L:
	for {
		select {
		case msg := <-client.C:
			w.Write([]byte(msg.UserData + "\n"))
			w.(http.Flusher).Flush()

			lastSent = time.Now()

		case <-ticker.C:
			if lastSent.Add(4 * time.Second).Before(time.Now()) {
				w.Write([]byte(":uhaha\n"))
				w.(http.Flusher).Flush()

				lastSent = time.Now()
			}

		case <-r.Context().Done():
			break L
		}
	}
}

func (self *Server) HandleProxy(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Auth-Key")), []byte(self.authKey)) == 0 {
		log.Error("Connection is closed because it has no valid auth key")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	remoteUrl, err := url.Parse("http://127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}

	httputil.NewSingleHostReverseProxy(remoteUrl).ServeHTTP(w, r)
}

func (self *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", self.HandleProxy)
	mux.HandleFunc("/monitor", self.HandleMonitor)

	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	self.server = &http.Server{
		Addr:         self.addr,
		Handler:      mux,
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return self.server.ListenAndServeTLS("/opt/premises/server.crt", "/opt/premises/server.key")
}

func (self *Server) Close() error {
	return self.server.Close()
}
