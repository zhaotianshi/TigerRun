# 老虎快跑

[English](README.md) | [中文](README.zh-CN.md)

老虎快跑是一个 Windows 桌面端 HTTP/HTTPS 抓包调试代理。它提供统一代理地址给浏览器、Windows 软件、iOS/Android Wi-Fi 代理使用，并内置本地 Root CA、HTTPS CONNECT 解密、请求/响应查看、快捷复制和 HAR 导出。

## 功能

- HTTP/HTTPS 代理服务，默认监听 `0.0.0.0:8080`
- HTTPS MITM：本地 Root CA + 动态域名证书
- 证书下载页：`http://<电脑IP>:8080/cert`
- Windows 当前用户系统代理一键开启/关闭
- Windows 当前用户根证书安装
- Chrome 扩展一键设置浏览器代理
- 上游代理链，兼容本地代理网关和企业代理
- 请求列表、方法筛选、搜索、详情查看
- 请求/响应 Headers、Body、完整报文、cURL 一键复制
- HAR 导出，并可直接打开导出位置
- TUN/VPN 模式架构预留

## 适用场景

- 调试网站、桌面软件、移动端 App 的 HTTP/HTTPS 请求
- 排查接口参数、Cookie、Headers、响应体和状态码
- 只让 Chrome 走抓包代理，不影响系统代理
- 在需要上游代理转发的环境下继续抓包
- 给测试、开发、接口联调导出 HAR 或复制 cURL

## 下载

编译好的安装包放在 GitHub Releases：

- [下载 Windows x64 版本](https://github.com/zhaotianshi/TigerRun/releases/latest/download/TigerRun-windows-amd64.zip)
- [查看全部发布包](https://github.com/zhaotianshi/TigerRun/releases)

目前只发布 Windows 构建。macOS 和 Linux 需要分别适配系统代理、证书安装和打包流程。

## 快速开始

本机浏览器或 Windows 软件：

```text
代理地址: 127.0.0.1
代理端口: 8080
```

iOS / Android：

```text
Wi-Fi 代理: <电脑局域网IP>:8080
证书下载: http://<电脑局域网IP>:8080/cert
```

## Chrome 扩展

如果只想让 Chrome 走抓包代理：

1. 打开 `chrome://extensions/`
2. 开启“开发者模式”
3. 加载 [chrome-extension](chrome-extension) 目录
4. 在扩展弹窗里点击“开启 Chrome 代理”

这样不会修改 Windows 系统代理，抓到的数据仍然显示在老虎快跑桌面后台。

## HTTPS 证书

HTTPS 解密需要设备信任老虎快跑生成的本地 Root CA。

### Windows

在桌面端点击“安装到 Windows”，证书会安装到当前用户的根证书库。

### iOS

1. 手机和电脑连接同一个 Wi-Fi
2. iPhone Wi-Fi 代理填 `<电脑IP>:8080`
3. 用 Safari 打开 `http://<电脑IP>:8080/cert`
4. 下载描述文件
5. 进入 `设置` -> `通用` -> `VPN与设备管理` 安装证书
6. 进入 `设置` -> `通用` -> `关于本机` -> `证书信任设置`
7. 开启 `老虎快跑 Local Debugging Root CA` 的完整信任

### Android

Android 6 及以下通常更容易信任用户安装的 CA。Android 7.0+ 默认不信任用户 CA，很多第三方 App 即使配置 Wi-Fi 代理，也只能看到连接，无法解密 HTTPS 明文。

如果是你自己的 Android App，可以在 debug 包里配置 `network_security_config` 允许用户 CA。若第三方 App 做了 SSL Pinning，通常需要 root、系统证书、调试包配置或 Frida/Objection 等额外方案。

## 上游代理

如果电脑需要通过另一个本地代理或企业代理转发流量，在左侧“上游代理”里填 HTTP 代理地址，例如：

```text
http://127.0.0.1:7890
```

链路：

```text
Chrome / Windows 软件 / 手机 -> 老虎快跑 -> 上游代理 -> Internet
```

这样老虎快跑仍然能抓包，上游代理继续负责后续连接。

## 从源码运行

依赖：

- Go `1.25+`
- Node.js `18+`
- npm
- Wails v2 CLI

```powershell
go test ./...

cd frontend
npm install
npm run build

cd ..
wails dev
```

## 构建 Windows exe

推荐使用项目脚本：

```powershell
.\scripts\build-windows.ps1
```

输出：

```text
build/bin/老虎快跑.exe
```

脚本会依次生成应用图标、前端静态资源、Windows `.syso` 资源，再构建 exe。脚本内的 `go build` 会带上 Wails 必需的 `production` build tag；不要直接裸跑 `go build`，否则启动时会出现 Wails build tags 错误。

## 图标

图标由 [tools/icongen](tools/icongen) 生成：

```powershell
go run ./tools/icongen
```

它会更新：

- `build/appicon.png`
- `build/windows/icon.ico`
- `frontend/src/assets/images/laohukuaipao-icon.png`
- `chrome-extension/icons/*.png`

## 项目结构

```text
.
├── app.go                     # Wails 后端绑定
├── internal/
│   ├── proxy/                 # HTTP/HTTPS 代理和 MITM 逻辑
│   ├── capture/               # 抓包会话存储和模型
│   ├── certstore/             # 本地 Root CA 和证书生成
│   ├── har/                   # HAR 导出
│   ├── localnet/              # 局域网地址发现
│   └── systemproxy/           # Windows 系统代理辅助函数
├── frontend/                  # React + TypeScript UI
├── chrome-extension/          # Chrome 代理切换扩展
├── tools/
│   ├── icongen/               # 图标生成器
│   └── winres/                # Windows 资源生成器
├── scripts/
│   └── build-windows.ps1      # Windows 构建脚本
└── docs/
    └── architecture.md        # 架构说明和 TUN/VPN 路线
```

## 架构

完整架构和 TUN/VPN 后续路线见 [docs/architecture.md](docs/architecture.md)。

## 限制

- 不支持绕过 App 的 SSL Pinning
- Android 7.0+ 第三方 App 默认不信任用户 CA
- 当前版本以 HTTP/HTTPS 代理模式为主，TUN/VPN 仍是架构预留
- 加密流量能否解密取决于目标软件是否信任本地 Root CA

## 安全提示

老虎快跑会生成本地 Root CA 用于调试 HTTPS 流量。请只在你自己的设备、授权测试环境或你有权调试的系统中使用。不要把生成的私钥分享给别人，也不要在生产环境长期信任调试 CA。

## License

MIT License. See [LICENSE](LICENSE).
