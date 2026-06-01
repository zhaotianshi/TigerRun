package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/zhaotianshi/TigerRun/internal/capture"
	"github.com/zhaotianshi/TigerRun/internal/certstore"
)

func TestProxyCapturesPlainHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	server, proxyURL, store := startTestProxy(t, false)
	defer stopTestProxy(t, server)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	resp, err := client.Get(backend.URL + "/v1/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
	sessions := waitForSummaries(t, store, 1)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 captured session, got %d", len(sessions))
	}
	if sessions[0].Method != http.MethodGet || sessions[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected session: %+v", sessions[0])
	}
}

func TestProxyServesCertificate(t *testing.T) {
	server, proxyURL, store := startTestProxy(t, false)
	defer stopTestProxy(t, server)

	resp, err := http.Get("http://" + proxyURL.Host + "/cert.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/x-x509-ca-cert" {
		t.Fatalf("unexpected content type: %s", got)
	}

	sessions := waitForSummaries(t, store, 1)
	if len(sessions) != 1 {
		t.Fatalf("expected certificate request to be captured, got %d sessions", len(sessions))
	}
	if sessions[0].Host != proxyURL.Host || sessions[0].Path != "/cert.pem" {
		t.Fatalf("unexpected certificate session: %+v", sessions[0])
	}
}

func TestProxyInterceptsHTTPS(t *testing.T) {
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("secure ok"))
	}))
	defer backend.Close()

	store := capture.NewStore()
	authority, err := certstore.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	addr := freeAddr(t)
	server := New(addr, store, authority)
	server.SetInterceptHTTPS(true)
	server.transportMu.Lock()
	server.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	server.transportMu.Unlock()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer stopTestProxy(t, server)

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(authority.CertPEM()) {
		t.Fatal("failed to append test CA")
	}
	proxyURL, _ := url.Parse("http://" + addr)
	client := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{RootCAs: caPool},
	}}

	resp, err := client.Get(backend.URL + "/secret")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "secure ok" {
		t.Fatalf("unexpected HTTPS body: %s", body)
	}

	sessions := waitForSummaries(t, store, 1)
	if len(sessions) != 1 || !sessions[0].InterceptedTLS {
		t.Fatalf("expected one intercepted HTTPS session, got %+v", sessions)
	}
}

func TestProxyUsesHTTPUpstreamForPlainHTTP(t *testing.T) {
	upstreamSawRequest := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamSawRequest = true
		if !r.URL.IsAbs() {
			t.Fatalf("upstream proxy expected absolute-form URL, got %q", r.URL.String())
		}
		if r.Host != "example.invalid" {
			t.Fatalf("unexpected upstream host: %s", r.Host)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("from upstream"))
	}))
	defer upstream.Close()

	server, proxyURL, store := startTestProxy(t, false)
	defer stopTestProxy(t, server)
	if err := server.SetUpstreamProxy(upstream.URL); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	resp, err := client.Get("http://example.invalid/via-upstream")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if !upstreamSawRequest {
		t.Fatal("upstream proxy did not receive the request")
	}
	if string(body) != "from upstream" {
		t.Fatalf("unexpected body: %s", body)
	}
	sessions := waitForSummaries(t, store, 1)
	if len(sessions) != 1 || sessions[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected captured sessions: %+v", sessions)
	}
}

func waitForSummaries(t *testing.T, store *capture.Store, count int) []capture.SessionSummary {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sessions := store.Summaries()
		if len(sessions) >= count {
			return sessions
		}
		time.Sleep(10 * time.Millisecond)
	}
	return store.Summaries()
}

func startTestProxy(t *testing.T, intercept bool) (*Server, *url.URL, *capture.Store) {
	t.Helper()
	store := capture.NewStore()
	authority, err := certstore.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	addr := freeAddr(t)
	server := New(addr, store, authority)
	server.SetInterceptHTTPS(intercept)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	proxyURL, _ := url.Parse("http://" + addr)
	return server, proxyURL, store
}

func stopTestProxy(t *testing.T, server *Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}
