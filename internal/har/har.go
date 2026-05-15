package har

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/zhaotianshi/TigerRun/internal/capture"
)

type logFile struct {
	Log logRoot `json:"log"`
}

type logRoot struct {
	Version string     `json:"version"`
	Creator creator    `json:"creator"`
	Entries []harEntry `json:"entries"`
}

type creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime time.Time   `json:"startedDateTime"`
	Time            int64       `json:"time"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         harTimings  `json:"timings"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
}

type harRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []nameValue  `json:"headers"`
	QueryString []nameValue  `json:"queryString"`
	Cookies     []nameValue  `json:"cookies"`
	HeadersSize int          `json:"headersSize"`
	BodySize    int64        `json:"bodySize"`
	PostData    *harPostData `json:"postData,omitempty"`
}

type harPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	Encoding string `json:"encoding,omitempty"`
}

type harResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []nameValue `json:"headers"`
	Cookies     []nameValue `json:"cookies"`
	Content     harContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
}

type harContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type harTimings struct {
	Send    int `json:"send"`
	Wait    int `json:"wait"`
	Receive int `json:"receive"`
}

type nameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Marshal(sessions []capture.SessionDetail) ([]byte, error) {
	entries := make([]harEntry, 0, len(sessions))
	for _, session := range sessions {
		entries = append(entries, toEntry(session))
	}

	return json.MarshalIndent(logFile{
		Log: logRoot{
			Version: "1.2",
			Creator: creator{
				Name:    "老虎快跑",
				Version: "0.1.0",
			},
			Entries: entries,
		},
	}, "", "  ")
}

func toEntry(session capture.SessionDetail) harEntry {
	startedAt, err := time.Parse(time.RFC3339Nano, session.StartedAt)
	if err != nil {
		startedAt = time.Now()
	}

	entry := harEntry{
		StartedDateTime: startedAt,
		Time:            session.DurationMs,
		Request: harRequest{
			Method:      session.Method,
			URL:         session.URL,
			HTTPVersion: session.Protocol,
			Headers:     headerList(session.RequestHeaders),
			QueryString: []nameValue{},
			Cookies:     []nameValue{},
			HeadersSize: -1,
			BodySize:    session.RequestBody.Size,
		},
		Response: harResponse{
			Status:      session.StatusCode,
			StatusText:  http.StatusText(session.StatusCode),
			HTTPVersion: session.Protocol,
			Headers:     headerList(session.ResponseHeaders),
			Cookies:     []nameValue{},
			Content: harContent{
				Size:     session.ResponseBody.Size,
				MimeType: session.ResponseBody.ContentType,
			},
			RedirectURL: headerFirst(session.ResponseHeaders, "Location"),
			HeadersSize: -1,
			BodySize:    session.ResponseBody.Size,
		},
		Timings: harTimings{
			Send:    0,
			Wait:    int(session.DurationMs),
			Receive: 0,
		},
		ServerIPAddress: session.Source,
	}

	if session.RequestBody.Text != "" {
		entry.Request.PostData = &harPostData{
			MimeType: session.RequestBody.ContentType,
			Text:     session.RequestBody.Text,
		}
	} else if session.RequestBody.Base64 != "" {
		entry.Request.PostData = &harPostData{
			MimeType: session.RequestBody.ContentType,
			Text:     session.RequestBody.Base64,
			Encoding: "base64",
		}
	}

	if session.ResponseBody.Text != "" && session.ResponseBody.Text != "[binary body]" {
		entry.Response.Content.Text = session.ResponseBody.Text
	} else if session.ResponseBody.Base64 != "" {
		entry.Response.Content.Text = session.ResponseBody.Base64
		entry.Response.Content.Encoding = "base64"
	} else {
		entry.Response.Content.Text = base64.StdEncoding.EncodeToString(nil)
	}

	return entry
}

func headerList(headers map[string][]string) []nameValue {
	out := make([]nameValue, 0)
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range headers[key] {
			out = append(out, nameValue{Name: key, Value: value})
		}
	}
	return out
}

func headerFirst(headers map[string][]string, key string) string {
	values := headers[http.CanonicalHeaderKey(key)]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
