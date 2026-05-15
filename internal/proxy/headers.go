package proxy

import "net/http"

var hopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Proxy-Connection":    {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func stripHopHeaders(headers http.Header) {
	for header := range hopHeaders {
		headers.Del(header)
	}
}

func cloneHeader(headers http.Header) http.Header {
	out := make(http.Header, len(headers))
	for key, values := range headers {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func writeHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if _, hop := hopHeaders[http.CanonicalHeaderKey(key)]; hop {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
