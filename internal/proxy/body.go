package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/zhaotianshi/TigerRun/internal/capture"
)

const maxBodyPreview = 1024 * 1024

func readBody(body io.ReadCloser, headers http.Header) (capture.BodyView, []byte, error) {
	if body == nil {
		return capture.BodyView{}, nil, nil
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return capture.BodyView{}, nil, err
	}

	return makeBodyView(raw, headers), raw, nil
}

func makeBodyView(raw []byte, headers http.Header) capture.BodyView {
	contentType := headers.Get("Content-Type")
	encoding := strings.ToLower(headers.Get("Content-Encoding"))
	previewBytes := raw

	if encoding == "gzip" {
		if decoded, err := gzip.NewReader(bytes.NewReader(raw)); err == nil {
			if data, readErr := io.ReadAll(io.LimitReader(decoded, maxBodyPreview+1)); readErr == nil {
				previewBytes = data
			}
			_ = decoded.Close()
		}
	}

	truncated := len(previewBytes) > maxBodyPreview
	if truncated {
		previewBytes = previewBytes[:maxBodyPreview]
	}

	view := capture.BodyView{
		Size:        int64(len(raw)),
		Truncated:   truncated,
		ContentType: contentType,
		Encoding:    encoding,
	}

	if isTextual(contentType, previewBytes) {
		view.Text = string(previewBytes)
		return view
	}

	if len(previewBytes) > 0 {
		view.Base64 = base64.StdEncoding.EncodeToString(previewBytes)
		view.Text = "[binary body]"
	}
	return view
}

func isTextual(contentType string, data []byte) bool {
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil {
			if strings.HasPrefix(mediaType, "text/") {
				return true
			}
			switch mediaType {
			case "application/json", "application/xml", "application/javascript", "application/x-www-form-urlencoded":
				return true
			}
			if strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml") {
				return true
			}
		}
	}
	if len(data) == 0 {
		return true
	}
	return utf8.Valid(data) && !bytes.Contains(data[:min(len(data), 256)], []byte{0})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
