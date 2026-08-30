// Telegram Bot API relay — a tiny reverse proxy to api.telegram.org.
//
// Our stage/prod VM is on RU-hosted Yandex Cloud where Telegram is fully
// filtered (both directions). This relay runs on Fly.io in a non-RU region,
// which reaches Telegram normally. The bot points TELEGRAM_API_BASE_URL at this
// app and every Bot API call (getUpdates, sendMessage, …) is forwarded verbatim.
//
//	bot ──► https://<app>.fly.dev/bot<token>/<method> ──► https://api.telegram.org/bot<token>/<method>
//
// Optional hardening: set RELAY_SECRET; the bot must then send the same value as
// the X-Relay-Secret header (backend: TELEGRAM_RELAY_SECRET) or requests are 403.
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func main() {
	target, err := url.Parse("https://api.telegram.org")
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	// Long-polling getUpdates blocks server-side (up to ~50s); give the upstream
	// transport generous timeouts so those requests are not cut off.
	proxy.Transport = &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 70 * time.Second,
	}
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.Host = target.Host // correct Host header + TLS SNI for Telegram
		r.Header.Del("X-Relay-Secret")
	}

	secret := os.Getenv("RELAY_SECRET")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret != "" && r.Header.Get("X-Relay-Secret") != secret {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if len(r.URL.Path) < 4 || r.URL.Path[:4] != "/bot" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// No read/write timeouts on the server: getUpdates long-polls and must not be
	// severed mid-wait; the upstream transport bounds the actual Telegram call.
	srv := &http.Server{Addr: ":" + port, Handler: handler}
	log.Printf("telegram relay listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
