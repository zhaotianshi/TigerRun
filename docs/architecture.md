# 老虎快跑架构说明

老虎快跑是一个桌面端 HTTP/HTTPS 调试代理。当前版本先交付代理模式：浏览器、Windows 软件、iOS/Android Wi-Fi 代理都指向同一个入口；TUN/VPN 模式作为后续入口接入到同一套抓包核心。

## 总体结构

```text
Windows 桌面应用
├─ Wails 桌面壳
│  ├─ React/TypeScript UI
│  └─ Go 后端绑定 API
├─ Chrome Extension
│  └─ 通过 chrome.proxy 把 Chrome 流量指向老虎快跑
├─ Ingress 入口
│  ├─ HTTP/HTTPS Proxy  已实现
│  └─ TUN/VPN Adapter   预留路线
├─ Capture Core
│  ├─ HTTP 代理转发
│  ├─ HTTPS CONNECT MITM
│  ├─ 上游代理链: 老虎快跑 -> 上游代理
│  ├─ 本地 Root CA + 动态站点证书
│  ├─ 请求/响应记录
│  └─ HAR 导出
└─ Platform Integration
   ├─ Windows 当前用户系统代理开关
   ├─ Windows 当前用户根证书安装
   └─ 局域网代理地址/证书下载页
```

## 端口和接入方式

默认代理端口是 `8080`。

```text
本机浏览器 / Windows 软件:
127.0.0.1:8080

iOS / Android / 局域网设备:
电脑局域网 IP:8080

证书下载:
http://电脑局域网 IP:8080/cert
```

桌面 UI 由 Wails 内嵌 WebView 提供，不需要单独暴露 Web 管理端口。开发期前端由 Vite 提供热更新。

如果本机已经使用另一个本地代理或企业代理，老虎快跑不应和它抢同一个浏览器代理位置，而应该串成代理链：

```text
Chrome / Windows 软件
        ↓
老虎快跑 127.0.0.1:8080
        ↓
上游代理 http://127.0.0.1:7890
        ↓
Internet
```

老虎快跑的上游代理支持：

```text
http://127.0.0.1:7890
https://127.0.0.1:7890
socks5://127.0.0.1:7890
```

HTTP/HTTPS 解密仍然发生在老虎快跑；上游代理只负责老虎快跑之后的出站连接。

Chrome 扩展位于 `chrome-extension`，它不直接读取请求内容，也不绕过浏览器安全模型。扩展只写入 Chrome 自身代理配置：

```text
chrome.proxy fixed_servers
singleProxy = http://127.0.0.1:8080
bypassList = <local>
```

因此 Chrome 的请求仍然进入同一个 `internal/proxy` 代理核心，桌面后台看到的数据和系统代理模式一致。

## HTTPS 解密流程

```text
客户端发起 CONNECT api.example.com:443
        ↓
老虎快跑返回 200 Connection Established
        ↓
老虎快跑使用本地 Root CA 动态签发 api.example.com 证书
        ↓
客户端与老虎快跑建立 TLS
        ↓
老虎快跑读取明文 HTTP 请求
        ↓
老虎快跑与真实 api.example.com 建立上游 TLS
        ↓
记录请求/响应，再把响应写回客户端
```

前提是客户端信任老虎快跑的 Root CA，且目标 App/软件没有 SSL Pinning。遇到证书锁定、私有加密协议、mTLS、代理检测时，代理模式和 TUN/VPN 模式都不能保证看到 HTTPS 明文。

## 当前代码分层

```text
app.go
  Wails 绑定 API，管理代理生命周期、证书、系统代理、HAR 导出。

internal/proxy
  HTTP/HTTPS 代理核心。处理普通 HTTP、CONNECT 隧道、HTTPS MITM、证书下载页、上游代理转发。

internal/certstore
  Root CA 生成/加载，按 Host 动态签发 TLS 证书。

internal/capture
  抓包记录模型和内存存储，供 UI 查询列表和详情。

internal/har
  将抓包记录导出为 HAR 1.2 JSON。

internal/localnet
  枚举局域网 IPv4 地址，供移动端配置代理。

internal/systemproxy
  Windows 当前用户系统代理开关；非 Windows 平台返回未实现。

frontend/src
  React 工作台界面：状态栏、接入向导、请求列表、详情面板。

chrome-extension
  Chrome Manifest V3 扩展：弹窗 UI、后台 service worker、浏览器代理设置、证书页入口。

tools/icongen
  生成老虎快跑应用图标、Windows ICO 和 Chrome 扩展图标。

tools/winres
  生成 Windows `.syso` 资源，把应用图标、窗口图标、exe 图标和版本信息写入最终可执行文件。
```

## TUN/VPN 扩展路线

TUN/VPN 不替代 HTTPS MITM，它只是把“不走 HTTP 代理”的流量导入老虎快跑。建议后续按平台分阶段实现：

```text
Capture Core
   ↑
HTTP Proxy Ingress         已实现
   ↑
TUN/VPN Ingress            待实现
   ├─ Windows: Wintun / WinDivert / WFP
   ├─ macOS: Network Extension / utun
   ├─ Linux: TUN/TAP + nftables
   ├─ Android: VpnService
   └─ iOS: Network Extension Packet Tunnel
```

Windows 桌面优先级建议：

1. 保持 HTTP/HTTPS 代理作为稳定基础。
2. 增加 Windows 透明代理入口：先评估 WinDivert，快速把 TCP 80/443 重定向到本地代理。
3. 再做 Wintun/WFP 版本，解决 DNS、IPv6、UDP、路由和权限问题。
4. 移动端 VPN 单独做 Android/iOS App，共用协议和 UI 设计，但不能直接由 Windows 桌面程序替代。

## 安全边界

- Root CA 私钥保存在当前用户配置目录下的 `老虎快跑/certs`。
- 证书只应安装在自己的调试设备上。
- 当前版本抓包记录保存在内存中，只有导出 HAR 时写入磁盘。
- Windows 系统代理修改的是当前用户注册表，不需要管理员权限。
- `certutil -user -addstore Root` 只安装到当前用户根证书库。

## 运行和构建

```powershell
# 后端测试
go test ./...

# 前端构建
npm --prefix frontend run build

# 重建图标和 Windows 资源
go run ./tools/icongen
go run ./tools/winres

# 桌面构建
go build -tags production -trimpath -ldflags "-H windowsgui" -o "build/bin/老虎快跑.exe" .
```

Windows 下也可以直接运行：

```powershell
.\scripts\build-windows.ps1
```

生成的可执行文件位于：

```text
build/bin/老虎快跑.exe
```
