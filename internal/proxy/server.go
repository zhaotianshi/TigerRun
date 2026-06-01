package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhaotianshi/TigerRun/internal/capture"
	"github.com/zhaotianshi/TigerRun/internal/certstore"
	socksproxy "golang.org/x/net/proxy"
)

type Server struct {
	addr      string
	store     *capture.Store
	authority *certstore.Authority

	transportMu sync.RWMutex
	transport   *http.Transport
	upstream    *upstreamConfig

	httpServer *http.Server
	running    atomic.Bool
	intercept  atomic.Bool
	mu         sync.Mutex
}

func New(addr string, store *capture.Store, authority *certstore.Authority) *Server {
	return &Server{
		addr:      addr,
		store:     store,
		authority: authority,
		transport: buildBaseTransport(),
	}
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return nil
	}

	server := &http.Server{
		Addr:              s.addr,
		Handler:           s,
		ReadHeaderTimeout: 30 * time.Second,
	}
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.httpServer = server
	s.running.Store(true)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.recordProxyError(err)
		}
		s.running.Store(false)
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil {
		s.running.Store(false)
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	s.httpServer = nil
	s.running.Store(false)
	return err
}

func (s *Server) SetInterceptHTTPS(enabled bool) {
	s.intercept.Store(enabled)
}

func (s *Server) InterceptHTTPS() bool {
	return s.intercept.Load()
}

func (s *Server) Running() bool {
	return s.running.Load()
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) SetUpstreamProxy(raw string) error {
	transport, upstream, err := buildTransport(strings.TrimSpace(raw))
	if err != nil {
		return err
	}

	s.transportMu.Lock()
	old := s.transport
	s.transport = transport
	s.upstream = upstream
	s.transportMu.Unlock()

	if old != nil {
		old.CloseIdleConnections()
	}
	return nil
}

func (s *Server) UpstreamProxy() string {
	s.transportMu.RLock()
	defer s.transportMu.RUnlock()
	if s.upstream == nil {
		return ""
	}
	return s.upstream.raw
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.isHealthRequest(r) {
		s.serveHealth(w)
		return
	}

	if s.isCertificateRequest(r) {
		s.recordCertificateRequest(r)
		s.serveCertificate(w, r)
		return
	}

	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
		return
	}

	s.forwardHTTP(w, r, "http", false)
}

func (s *Server) isHealthRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.Path)
	return path == "/__laohukuaipao/health" || path == "/laohukuaipao/health" || path == "/__packetlens/health" || path == "/packetlens/health"
}

func (s *Server) serveHealth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":           "老虎快跑",
		"ok":             true,
		"proxy":          s.Running(),
		"interceptHttps": s.InterceptHTTPS(),
		"upstreamProxy":  s.UpstreamProxy(),
		"captured":       s.store.Count(),
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !s.intercept.Load() {
		s.tunnelConnect(w, r)
		return
	}
	s.interceptConnect(w, r)
}

func (s *Server) tunnelConnect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	source := remoteHost(r.RemoteAddr)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	targetConn, err := s.dialTarget(ctx, r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		s.store.Add(&capture.CapturedSession{
			StartedAt:  start,
			FinishedAt: time.Now(),
			Source:     source,
			Method:     http.MethodConnect,
			Scheme:     "https",
			Host:       r.Host,
			Path:       r.Host,
			URL:        "https://" + r.Host,
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Protocol:   r.Proto,
			TunnelOnly: true,
			Error:      err.Error(),
		})
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking is not supported", http.StatusInternalServerError)
		_ = targetConn.Close()
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		_ = targetConn.Close()
		return
	}

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	s.store.Add(&capture.CapturedSession{
		StartedAt:  start,
		FinishedAt: time.Now(),
		Source:     source,
		Method:     http.MethodConnect,
		Scheme:     "https",
		Host:       r.Host,
		Path:       r.Host,
		URL:        "https://" + r.Host,
		StatusCode: http.StatusOK,
		Status:     "200 Connection Established",
		Protocol:   r.Proto,
		TunnelOnly: true,
	})

	go pipeAndClose(targetConn, clientConn)
	go pipeAndClose(clientConn, targetConn)
}

func (s *Server) interceptConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking is not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	host := hostWithoutPort(r.Host)
	cert, err := s.authority.CertificateFor(host)
	if err != nil {
		_ = clientConn.Close()
		s.recordProxyError(fmt.Errorf("issue certificate for %s: %w", host, err))
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
		NextProtos:   []string{"http/1.1"},
		MinVersion:   tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		s.store.Add(&capture.CapturedSession{
			StartedAt:      time.Now(),
			FinishedAt:     time.Now(),
			Source:         remoteHost(r.RemoteAddr),
			Method:         http.MethodConnect,
			Scheme:         "https",
			Host:           r.Host,
			Path:           r.Host,
			URL:            "https://" + r.Host,
			StatusCode:     495,
			Status:         "TLS handshake failed",
			Protocol:       r.Proto,
			InterceptedTLS: true,
			Error:          err.Error(),
		})
		return
	}

	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.recordProxyError(fmt.Errorf("read TLS request from %s: %w", r.Host, err))
			}
			_ = tlsConn.Close()
			return
		}

		if req.URL == nil {
			req.URL = &url.URL{}
		}
		if req.URL.Scheme == "" {
			req.URL.Scheme = "https"
		}
		if req.URL.Host == "" {
			req.URL.Host = r.Host
		}
		req.Host = r.Host
		req.RemoteAddr = r.RemoteAddr

		closeAfter := req.Close
		s.forwardIntercepted(tlsConn, req, closeAfter)
		if closeAfter {
			_ = tlsConn.Close()
			return
		}
	}
}

func (s *Server) forwardHTTP(w http.ResponseWriter, r *http.Request, scheme string, intercepted bool) {
	detail, resp, respBody, err := s.performRoundTrip(r, scheme, intercepted)
	if err != nil {
		status := http.StatusBadGateway
		http.Error(w, err.Error(), status)
		detail.FinishedAt = time.Now()
		detail.StatusCode = status
		detail.Status = "502 Bad Gateway"
		detail.Error = err.Error()
		s.store.Add(detail)
		return
	}
	defer resp.Body.Close()

	writeHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	detail.FinishedAt = time.Now()
	detail.StatusCode = resp.StatusCode
	detail.Status = resp.Status
	detail.Protocol = resp.Proto
	detail.ResponseHeaders = cloneHeader(resp.Header)
	detail.ResponseBody = makeBodyView(respBody, resp.Header)
	detail.ResponseSize = int64(len(respBody))
	detail.ContentType = resp.Header.Get("Content-Type")
	s.store.Add(detail)
}

func (s *Server) forwardIntercepted(conn net.Conn, req *http.Request, closeAfter bool) {
	detail, resp, respBody, err := s.performRoundTrip(req, "https", true)
	if err != nil {
		detail.FinishedAt = time.Now()
		detail.StatusCode = http.StatusBadGateway
		detail.Status = "502 Bad Gateway"
		detail.Error = err.Error()
		s.store.Add(detail)
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + err.Error()))
		return
	}
	defer resp.Body.Close()

	if closeAfter {
		resp.Close = true
		resp.Header.Set("Connection", "close")
	}
	upstreamProto := resp.Proto
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	resp.ContentLength = int64(len(respBody))
	resp.Proto = "HTTP/1.1"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1
	if err := resp.Write(conn); err != nil {
		detail.Error = err.Error()
	}

	detail.FinishedAt = time.Now()
	detail.StatusCode = resp.StatusCode
	detail.Status = resp.Status
	detail.Protocol = upstreamProto
	detail.ResponseHeaders = cloneHeader(resp.Header)
	detail.ResponseBody = makeBodyView(respBody, resp.Header)
	detail.ResponseSize = int64(len(respBody))
	detail.ContentType = resp.Header.Get("Content-Type")
	detail.Certificate = capture.CertificateInfo{
		ServerName: detail.Host,
		Issuer:     s.authority.RootSubject(),
		Subject:    detail.Host,
		NotAfter:   s.authority.RootExpires().Format(time.RFC3339),
	}
	s.store.Add(detail)
}

func (s *Server) performRoundTrip(r *http.Request, fallbackScheme string, intercepted bool) (*capture.CapturedSession, *http.Response, []byte, error) {
	start := time.Now()
	detail := &capture.CapturedSession{
		StartedAt:      start,
		Source:         remoteHost(r.RemoteAddr),
		Method:         r.Method,
		Scheme:         fallbackScheme,
		Host:           r.Host,
		Path:           requestPath(r),
		URL:            requestURL(r, fallbackScheme),
		Protocol:       r.Proto,
		InterceptedTLS: intercepted,
		RequestHeaders: cloneHeader(r.Header),
	}

	reqBody, rawBody, err := readBody(r.Body, r.Header)
	if err != nil {
		return detail, nil, nil, err
	}
	detail.RequestBody = reqBody
	detail.RequestSize = int64(len(rawBody))

	outReq := r.Clone(context.Background())
	outReq.RequestURI = ""
	outReq.Body = io.NopCloser(bytes.NewReader(rawBody))
	outReq.ContentLength = int64(len(rawBody))
	outReq.Header = cloneHeader(r.Header)
	stripHopHeaders(outReq.Header)
	if outReq.URL == nil {
		outReq.URL = &url.URL{}
	}
	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = fallbackScheme
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = r.Host
	}
	outReq.Host = r.Host

	transport := s.currentTransport()
	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		return detail, nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return detail, resp, nil, err
	}
	return detail, resp, respBody, nil
}

func (s *Server) currentTransport() *http.Transport {
	s.transportMu.RLock()
	defer s.transportMu.RUnlock()
	return s.transport
}

func (s *Server) currentUpstream() *upstreamConfig {
	s.transportMu.RLock()
	defer s.transportMu.RUnlock()
	if s.upstream == nil {
		return nil
	}
	cp := *s.upstream
	return &cp
}

func (s *Server) dialTarget(ctx context.Context, target string) (net.Conn, error) {
	upstream := s.currentUpstream()
	if upstream == nil {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "tcp", target)
	}

	switch upstream.kind {
	case "http":
		return dialViaHTTPProxy(ctx, upstream, target)
	case "socks5":
		return dialViaSOCKSProxy(ctx, upstream, target)
	default:
		return nil, fmt.Errorf("unsupported upstream proxy %q", upstream.kind)
	}
}

func buildTransport(raw string) (*http.Transport, *upstreamConfig, error) {
	transport := buildBaseTransport()
	if raw == "" {
		return transport, nil, nil
	}

	upstream, err := parseUpstreamProxy(raw)
	if err != nil {
		return nil, nil, err
	}

	switch upstream.kind {
	case "http":
		transport.Proxy = http.ProxyURL(upstream.url)
	case "socks5":
		dialer, err := socksproxy.SOCKS5("tcp", upstream.address, nil, socksproxy.Direct)
		if err != nil {
			return nil, nil, err
		}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			type result struct {
				conn net.Conn
				err  error
			}
			done := make(chan result, 1)
			go func() {
				conn, err := dialer.Dial(network, address)
				done <- result{conn: conn, err: err}
			}()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case result := <-done:
				return result.conn, result.err
			}
		}
	default:
		return nil, nil, fmt.Errorf("unsupported upstream proxy %q", upstream.kind)
	}

	return transport, upstream, nil
}

func buildBaseTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

type upstreamConfig struct {
	raw     string
	kind    string
	url     *url.URL
	address string
}

func parseUpstreamProxy(raw string) (*upstreamConfig, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, errors.New("上游代理地址缺少 host")
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return &upstreamConfig{raw: parsed.String(), kind: "http", url: parsed, address: parsed.Host}, nil
	case "socks5", "socks5h":
		return &upstreamConfig{raw: parsed.String(), kind: "socks5", url: parsed, address: parsed.Host}, nil
	default:
		return nil, fmt.Errorf("上游代理只支持 http、https、socks5: %s", parsed.Scheme)
	}
}

func dialViaHTTPProxy(ctx context.Context, upstream *upstreamConfig, target string) (net.Conn, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", upstream.address)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(upstream.url.Scheme, "https") {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: hostWithoutPort(upstream.address), MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	req := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n"
	if auth := proxyAuthorization(upstream.url); auth != "" {
		req += "Proxy-Authorization: " + auth + "\r\n"
	}
	req += "\r\n"

	if _, err := io.WriteString(conn, req); err != nil {
		_ = conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_ = conn.Close()
		return nil, fmt.Errorf("upstream proxy CONNECT failed: %s", resp.Status)
	}
	return conn, nil
}

func dialViaSOCKSProxy(ctx context.Context, upstream *upstreamConfig, target string) (net.Conn, error) {
	dialer, err := socksproxy.SOCKS5("tcp", upstream.address, nil, socksproxy.Direct)
	if err != nil {
		return nil, err
	}
	type result struct {
		conn net.Conn
		err  error
	}
	done := make(chan result, 1)
	go func() {
		conn, err := dialer.Dial("tcp", target)
		done <- result{conn: conn, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		return result.conn, result.err
	}
}

func proxyAuthorization(proxyURL *url.URL) string {
	if proxyURL.User == nil {
		return ""
	}
	password, _ := proxyURL.User.Password()
	token := proxyURL.User.Username() + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}

func (s *Server) isCertificateRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.Path)
	if path != "/cert" && path != "/cert.pem" && path != "/ca.pem" && path != "/laohukuaipao-root-ca.pem" && path != "/packetlens-root-ca.pem" && path != "/" {
		return false
	}

	host := strings.ToLower(hostWithoutPort(r.Host))
	if host == "laohukuaipao.local" || host == "laohukuaipao.cert" || host == "packetlens.local" || host == "packetlens.cert" || host == "mitm.it" || host == "localhost" || host == "127.0.0.1" {
		return true
	}

	if net.ParseIP(host) != nil && (path == "/cert" || path == "/cert.pem" || path == "/ca.pem" || path == "/laohukuaipao-root-ca.pem" || path == "/packetlens-root-ca.pem") {
		return true
	}

	return r.URL.Scheme == "" && (path == "/" || path == "/cert" || path == "/cert.pem" || path == "/ca.pem" || path == "/laohukuaipao-root-ca.pem" || path == "/packetlens-root-ca.pem")
}

func (s *Server) serveCertificate(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>老虎快跑 Certificate</title><style>body{font-family:Segoe UI,Arial,sans-serif;max-width:760px;margin:48px auto;color:#1f2937;line-height:1.6}a{color:#0f766e}code{background:#f3f4f6;padding:2px 6px;border-radius:4px}</style></head><body><h1>老虎快跑调试证书</h1><p>下载并信任根证书后，老虎快跑才能解密 HTTPS 流量。只在你自己的设备和调试环境中使用。</p><p><a href="/cert.pem">下载根证书 laohukuaipao-root-ca.pem</a></p><p>iOS 安装后还需要进入 <code>设置 - 通用 - 关于本机 - 证书信任设置</code> 开启完整信任。</p></body></html>`))
		return
	}

	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="laohukuaipao-root-ca.pem"`)
	_, _ = w.Write(s.authority.CertPEM())
}

func (s *Server) recordCertificateRequest(r *http.Request) {
	contentType := "application/x-x509-ca-cert"
	if r.URL.Path == "/" || strings.EqualFold(r.URL.Path, "/cert") {
		contentType = "text/html; charset=utf-8"
	}
	s.store.Add(&capture.CapturedSession{
		StartedAt:       time.Now(),
		FinishedAt:      time.Now(),
		Source:          remoteHost(r.RemoteAddr),
		Method:          r.Method,
		Scheme:          "http",
		Host:            r.Host,
		Path:            requestPath(r),
		URL:             requestURL(r, "http"),
		StatusCode:      http.StatusOK,
		Status:          "200 OK",
		Protocol:        r.Proto,
		ContentType:     contentType,
		RequestHeaders:  cloneHeader(r.Header),
		ResponseHeaders: map[string][]string{"Content-Type": {contentType}},
		Rule:            "internal certificate",
	})
}

func (s *Server) recordProxyError(err error) {
	s.store.Add(&capture.CapturedSession{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Method:     "PROXY",
		StatusCode: 0,
		Status:     "Proxy error",
		Error:      err.Error(),
	})
}

func pipeAndClose(dst net.Conn, src net.Conn) {
	_, _ = io.Copy(dst, src)
	_ = dst.Close()
	_ = src.Close()
}

func requestPath(r *http.Request) string {
	if r.URL == nil {
		return ""
	}
	path := r.URL.RequestURI()
	if path == "" {
		path = r.URL.Path
	}
	return path
}

func requestURL(r *http.Request, fallbackScheme string) string {
	if r.URL == nil {
		return fallbackScheme + "://" + r.Host
	}
	if r.URL.IsAbs() {
		return r.URL.String()
	}
	return fallbackScheme + "://" + r.Host + r.URL.RequestURI()
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func hostWithoutPort(hostport string) string {
	if hostport == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(hostport)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	if strings.Count(hostport, ":") == 1 {
		return strings.Split(hostport, ":")[0]
	}
	return strings.Trim(hostport, "[]")
}
