package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/zhaotianshi/TigerRun/internal/capture"
	"github.com/zhaotianshi/TigerRun/internal/certstore"
	"github.com/zhaotianshi/TigerRun/internal/har"
	"github.com/zhaotianshi/TigerRun/internal/localnet"
	"github.com/zhaotianshi/TigerRun/internal/proxy"
	"github.com/zhaotianshi/TigerRun/internal/systemproxy"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultProxyPort = 8080
	appVersion       = "0.1.0"
)

type App struct {
	ctx       context.Context
	store     *capture.Store
	authority *certstore.Authority
	proxy     *proxy.Server
	port      int
	dataDir   string
	settings  Settings
}

type Settings struct {
	UpstreamProxy string `json:"upstreamProxy"`
}

type Status struct {
	Version              string   `json:"version"`
	ProxyRunning         bool     `json:"proxyRunning"`
	InterceptHTTPS       bool     `json:"interceptHttps"`
	ProxyPort            int      `json:"proxyPort"`
	LocalProxyAddress    string   `json:"localProxyAddress"`
	LANProxyAddresses    []string `json:"lanProxyAddresses"`
	MobileProxyHost      string   `json:"mobileProxyHost"`
	MobileProxyPort      int      `json:"mobileProxyPort"`
	CertificateURL       string   `json:"certificateUrl"`
	CertificatePath      string   `json:"certificatePath"`
	CertificateSubject   string   `json:"certificateSubject"`
	CertificateExpiresAt string   `json:"certificateExpiresAt"`
	SessionCount         int      `json:"sessionCount"`
	SystemProxyHint      string   `json:"systemProxyHint"`
	TUNStatus            string   `json:"tunStatus"`
	UpstreamProxy        string   `json:"upstreamProxy"`
}

func NewApp() *App {
	dataDir := appDataDir()
	ca, err := certstore.LoadOrCreate(filepath.Join(dataDir, "certs"))
	if err != nil {
		panic(err)
	}

	store := capture.NewStore()
	p := proxy.New(fmt.Sprintf("0.0.0.0:%d", defaultProxyPort), store, ca)
	p.SetInterceptHTTPS(false)
	settings := loadSettings(dataDir)
	if settings.UpstreamProxy != "" {
		_ = p.SetUpstreamProxy(settings.UpstreamProxy)
	}

	return &App{
		store:     store,
		authority: ca,
		proxy:     p,
		port:      defaultProxyPort,
		dataDir:   dataDir,
		settings:  settings,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = os.MkdirAll(a.dataDir, 0o755)
	_ = a.proxy.Start()
}

func (a *App) shutdown(ctx context.Context) {
	if a.proxy != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = a.proxy.Stop(stopCtx)
	}
}

func (a *App) GetStatus() Status {
	lanAddrs := localnet.LANAddresses()
	lanProxyAddrs := make([]string, 0, len(lanAddrs))
	for _, addr := range lanAddrs {
		lanProxyAddrs = append(lanProxyAddrs, fmt.Sprintf("%s:%d", addr, a.port))
	}

	mobileProxyHost := ""
	certURL := ""
	if len(lanAddrs) > 0 {
		mobileProxyHost = lanAddrs[0]
		certURL = fmt.Sprintf("http://%s:%d/cert", lanAddrs[0], a.port)
	} else {
		mobileProxyHost = "127.0.0.1"
		certURL = fmt.Sprintf("http://127.0.0.1:%d/cert", a.port)
	}

	return Status{
		Version:              appVersion,
		ProxyRunning:         a.proxy.Running(),
		InterceptHTTPS:       a.proxy.InterceptHTTPS(),
		ProxyPort:            a.port,
		LocalProxyAddress:    fmt.Sprintf("127.0.0.1:%d", a.port),
		LANProxyAddresses:    lanProxyAddrs,
		MobileProxyHost:      mobileProxyHost,
		MobileProxyPort:      a.port,
		CertificateURL:       certURL,
		CertificatePath:      a.authority.CertPath(),
		CertificateSubject:   a.authority.RootSubject(),
		CertificateExpiresAt: a.authority.RootExpires().Format("2006-01-02"),
		SessionCount:         a.store.Count(),
		SystemProxyHint:      "Windows 系统代理会写入当前用户设置，不需要管理员权限。",
		TUNStatus:            "已预留架构接口；当前版本先交付 HTTP/HTTPS 代理模式。",
		UpstreamProxy:        a.proxy.UpstreamProxy(),
	}
}

func (a *App) StartProxy() error {
	return a.proxy.Start()
}

func (a *App) StopProxy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return a.proxy.Stop(ctx)
}

func (a *App) SetHTTPSIntercept(enabled bool) {
	a.proxy.SetInterceptHTTPS(enabled)
}

func (a *App) SetUpstreamProxy(raw string) error {
	if err := a.proxy.SetUpstreamProxy(raw); err != nil {
		return err
	}
	a.settings.UpstreamProxy = a.proxy.UpstreamProxy()
	return a.saveSettings()
}

func (a *App) ClearUpstreamProxy() error {
	if err := a.proxy.SetUpstreamProxy(""); err != nil {
		return err
	}
	a.settings.UpstreamProxy = ""
	return a.saveSettings()
}

func (a *App) EnableSystemProxy() error {
	if !a.proxy.Running() {
		if err := a.proxy.Start(); err != nil {
			return err
		}
	}
	return systemproxy.Enable(fmt.Sprintf("127.0.0.1:%d", a.port))
}

func (a *App) DisableSystemProxy() error {
	return systemproxy.Disable()
}

func (a *App) AllowMobileFirewallAccess() error {
	if runtime.GOOS != "windows" {
		return errors.New("当前版本只支持在 Windows 上创建防火墙规则")
	}
	ruleName := fmt.Sprintf("TigerRun HTTP Proxy %d", a.port)
	innerScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$name = %s
$port = %d
$rule = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Select-Object -First 1
if ($null -eq $rule) {
  New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort $port -Profile Any | Out-Null
} else {
  Set-NetFirewallRule -DisplayName $name -Enabled True -Action Allow -Profile Any
  Get-NetFirewallPortFilter -AssociatedNetFirewallRule $rule | Set-NetFirewallPortFilter -Protocol TCP -LocalPort $port
}
`, powerShellString(ruleName), a.port)
	outerScript := fmt.Sprintf(`$p = Start-Process -FilePath 'powershell.exe' -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-EncodedCommand','%s') -Verb RunAs -Wait -PassThru; if ($p.ExitCode -ne 0) { throw "Firewall rule command failed with exit code $($p.ExitCode)" }`, powerShellEncodedCommand(innerScript))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", outerScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建 Windows 防火墙入站规则失败: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *App) InstallRootCertificate() error {
	if runtime.GOOS != "windows" {
		return errors.New("当前版本只实现 Windows 当前用户证书安装")
	}
	cmd := exec.Command("certutil", "-user", "-addstore", "Root", a.authority.CertPath())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func (a *App) OpenCertificateLocation() error {
	if runtime.GOOS == "windows" {
		return exec.Command("explorer.exe", "/select,", a.authority.CertPath()).Start()
	}
	wailsRuntime.BrowserOpenURL(a.ctx, "file://"+filepath.Dir(a.authority.CertPath()))
	return nil
}

func (a *App) OpenPathLocation(path string) error {
	if path == "" {
		return errors.New("文件路径为空")
	}
	cleanPath := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
			return exec.Command("explorer.exe", cleanPath).Start()
		}
		return exec.Command("explorer.exe", "/select,", cleanPath).Start()
	}
	wailsRuntime.BrowserOpenURL(a.ctx, "file://"+filepath.Dir(cleanPath))
	return nil
}

func (a *App) OpenCertificateURL() error {
	wailsRuntime.BrowserOpenURL(a.ctx, a.GetStatus().CertificateURL)
	return nil
}

func (a *App) GetSessions() []capture.SessionSummary {
	return a.store.Summaries()
}

func (a *App) GetSession(id string) (capture.SessionDetail, error) {
	session, ok := a.store.Detail(id)
	if !ok {
		return capture.SessionDetail{}, fmt.Errorf("session %s not found", id)
	}
	return session, nil
}

func (a *App) ClearSessions() {
	a.store.Clear()
}

func (a *App) ExportHAR() (string, error) {
	data, err := har.Marshal(a.store.AllDetails())
	if err != nil {
		return "", err
	}
	outDir := filepath.Join(a.dataDir, "exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(outDir, fmt.Sprintf("laohukuaipao-%s.har", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) CopyProxyAddress() error {
	return wailsRuntime.ClipboardSetText(a.ctx, fmt.Sprintf("127.0.0.1:%d", a.port))
}

func (a *App) CopyCertificateURL() error {
	return wailsRuntime.ClipboardSetText(a.ctx, a.GetStatus().CertificateURL)
}

func (a *App) saveSettings() error {
	if err := os.MkdirAll(a.dataDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(a.dataDir), data, 0o600)
}

func loadSettings(dataDir string) Settings {
	data, err := os.ReadFile(settingsPath(dataDir))
	if err != nil {
		return Settings{}
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}
	}
	return settings
}

func settingsPath(dataDir string) string {
	return filepath.Join(dataDir, "settings.json")
}

func appDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "老虎快跑")
}

func isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func powerShellString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func powerShellEncodedCommand(script string) string {
	wide := utf16.Encode([]rune(script))
	raw := make([]byte, len(wide)*2)
	for i, code := range wide {
		raw[i*2] = byte(code)
		raw[i*2+1] = byte(code >> 8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
