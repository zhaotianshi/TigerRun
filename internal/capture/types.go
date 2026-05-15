package capture

import "time"

type SessionSummary struct {
	ID              string `json:"id"`
	StartedAt       string `json:"startedAt"`
	DurationMs      int64  `json:"durationMs"`
	Source          string `json:"source"`
	Method          string `json:"method"`
	Scheme          string `json:"scheme"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	URL             string `json:"url"`
	StatusCode      int    `json:"statusCode"`
	Status          string `json:"status"`
	Protocol        string `json:"protocol"`
	ContentType     string `json:"contentType"`
	ResponseSize    int64  `json:"responseSize"`
	RequestSize     int64  `json:"requestSize"`
	InterceptedTLS  bool   `json:"interceptedTls"`
	TunnelOnly      bool   `json:"tunnelOnly"`
	Error           string `json:"error"`
	Rule            string `json:"rule"`
	ResponsePreview string `json:"responsePreview"`
}

type SessionDetail struct {
	SessionSummary
	RequestHeaders  map[string][]string `json:"requestHeaders"`
	ResponseHeaders map[string][]string `json:"responseHeaders"`
	RequestBody     BodyView            `json:"requestBody"`
	ResponseBody    BodyView            `json:"responseBody"`
	Timing          Timing              `json:"timing"`
	Certificate     CertificateInfo     `json:"certificate"`
}

type BodyView struct {
	Text        string `json:"text"`
	Base64      string `json:"base64"`
	Size        int64  `json:"size"`
	Truncated   bool   `json:"truncated"`
	ContentType string `json:"contentType"`
	Encoding    string `json:"encoding"`
}

type Timing struct {
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	DurationMs int64  `json:"durationMs"`
}

type CertificateInfo struct {
	ServerName string `json:"serverName"`
	Issuer     string `json:"issuer"`
	Subject    string `json:"subject"`
	NotBefore  string `json:"notBefore"`
	NotAfter   string `json:"notAfter"`
}

type CapturedSession struct {
	ID              string
	StartedAt       time.Time
	FinishedAt      time.Time
	Source          string
	Method          string
	Scheme          string
	Host            string
	Path            string
	URL             string
	StatusCode      int
	Status          string
	Protocol        string
	ContentType     string
	ResponseSize    int64
	RequestSize     int64
	InterceptedTLS  bool
	TunnelOnly      bool
	Error           string
	Rule            string
	RequestHeaders  map[string][]string
	ResponseHeaders map[string][]string
	RequestBody     BodyView
	ResponseBody    BodyView
	Timing          Timing
	Certificate     CertificateInfo
}
