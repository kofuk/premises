package outbound

import (
	"crypto/subtle"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kofuk/premises/exteriord/msgrouter"
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
		log.Println("Connection is closed because it has no valid auth key")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	msgChan := self.msgRouter.Subscribe()
	defer self.msgRouter.Unsubscribe(msgChan)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastSent := time.Now().Add(time.Hour)

L:
	for {
		select {
		case msg := <-msgChan:
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
		log.Println("Connection is closed because it has no valid auth key")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	r.URL.Scheme = "http"
	r.URL.Host = "127.0.0.1:9000"
	r.RequestURI = ""

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Println("Unable to proxy incoming request:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	io.ReadAll(resp.Body)
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
