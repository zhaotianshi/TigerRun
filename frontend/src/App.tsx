import {useEffect, useMemo, useRef, useState} from 'react';
import type {ReactNode} from 'react';
import {
  BadgeCheck,
  Ban,
  Braces,
  CheckCircle2,
  Circle,
  Copy,
  Download,
  Eye,
  FileDown,
  Globe2,
  HardDrive,
  KeyRound,
  Laptop,
  ListFilter,
  Network,
  Play,
  Power,
  RefreshCcw,
  Search,
  ShieldCheck,
  Smartphone,
  Trash2,
  Wifi,
} from 'lucide-react';
import './App.css';
import {
  ClearSessions,
  CopyCertificateURL,
  CopyProxyAddress,
  DisableSystemProxy,
  EnableSystemProxy,
  ExportHAR,
  GetSession,
  GetSessions,
  GetStatus,
  InstallRootCertificate,
  OpenCertificateLocation,
  OpenCertificateURL,
  OpenPathLocation,
  ClearUpstreamProxy,
  SetHTTPSIntercept,
  SetUpstreamProxy,
  StartProxy,
  StopProxy,
} from '../wailsjs/go/main/App';
import {ClipboardSetText} from '../wailsjs/runtime/runtime';
import appIcon from './assets/images/laohukuaipao-icon.png';

type Status = {
  version: string;
  proxyRunning: boolean;
  interceptHttps: boolean;
  proxyPort: number;
  localProxyAddress: string;
  lanProxyAddresses: string[];
  certificateUrl: string;
  certificatePath: string;
  certificateSubject: string;
  certificateExpiresAt: string;
  sessionCount: number;
  systemProxyHint: string;
  tunStatus: string;
  upstreamProxy: string;
};

type SessionSummary = {
  id: string;
  startedAt: string;
  durationMs: number;
  source: string;
  method: string;
  scheme: string;
  host: string;
  path: string;
  url: string;
  statusCode: number;
  status: string;
  protocol: string;
  contentType: string;
  responseSize: number;
  requestSize: number;
  interceptedTls: boolean;
  tunnelOnly: boolean;
  error: string;
  rule: string;
  responsePreview: string;
};

type BodyView = {
  text: string;
  base64: string;
  size: number;
  truncated: boolean;
  contentType: string;
  encoding: string;
};

type SessionDetail = SessionSummary & {
  requestHeaders: Record<string, string[]>;
  responseHeaders: Record<string, string[]>;
  requestBody: BodyView;
  responseBody: BodyView;
  timing: {
    startedAt: string;
    finishedAt: string;
    durationMs: number;
  };
  certificate: {
    serverName: string;
    issuer: string;
    subject: string;
    notBefore: string;
    notAfter: string;
  };
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: unknown;
      };
    };
  }
}

const emptyStatus: Status = {
  version: '0.1.0',
  proxyRunning: false,
  interceptHttps: true,
  proxyPort: 8080,
  localProxyAddress: '127.0.0.1:8080',
  lanProxyAddresses: [],
  certificateUrl: 'http://127.0.0.1:8080/cert',
  certificatePath: '',
  certificateSubject: '',
  certificateExpiresAt: '',
  sessionCount: 0,
  systemProxyHint: '',
  tunStatus: '',
  upstreamProxy: '',
};

function App() {
  const [status, setStatus] = useState<Status>(emptyStatus);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string>('');
  const [detail, setDetail] = useState<SessionDetail | null>(null);
  const [query, setQuery] = useState('');
  const [methodFilter, setMethodFilter] = useState('ALL');
  const [upstreamDraft, setUpstreamDraft] = useState('');
  const [editingUpstream, setEditingUpstream] = useState(false);
  const [detailTab, setDetailTab] = useState<'overview' | 'request' | 'response' | 'cert'>('overview');
  const [notice, setNotice] = useState('');
  const [noticePath, setNoticePath] = useState('');
  const selectedIdRef = useRef('');
  const editingUpstreamRef = useRef(false);
  const noticeTimerRef = useRef<number>();

  async function refresh() {
    if (!hasWailsRuntime()) {
      return;
    }
    const [nextStatus, nextSessions] = await Promise.all([GetStatus(), GetSessions()]);
    setStatus(nextStatus as Status);
    setSessions(nextSessions as SessionSummary[]);
    if (!editingUpstreamRef.current) {
      setUpstreamDraft((nextStatus as Status).upstreamProxy || '');
    }

    const currentSelected = selectedIdRef.current;
    const currentStillExists = nextSessions.some((item) => item.id === currentSelected);
    if (currentSelected && currentStillExists) {
      return;
    }

    const nextSelected = nextSessions[0]?.id || '';
    if (nextSelected !== currentSelected) {
      selectedIdRef.current = nextSelected;
      setSelectedId(nextSelected);
    }
  }

  useEffect(() => {
    refresh().catch(showError);
    const timer = window.setInterval(() => refresh().catch(showError), 1200);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    selectedIdRef.current = selectedId;
  }, [selectedId]);

  useEffect(() => {
    editingUpstreamRef.current = editingUpstream;
  }, [editingUpstream]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      return;
    }
    GetSession(selectedId).then((value) => setDetail(value as SessionDetail)).catch(() => setDetail(null));
  }, [selectedId, sessions.length]);

  useEffect(() => {
    if (noticeTimerRef.current) {
      window.clearTimeout(noticeTimerRef.current);
      noticeTimerRef.current = undefined;
    }
    if (!notice) {
      return;
    }
    noticeTimerRef.current = window.setTimeout(() => {
      setNotice('');
      setNoticePath('');
      noticeTimerRef.current = undefined;
    }, noticePath ? 15000 : 4500);
    return () => {
      if (noticeTimerRef.current) {
        window.clearTimeout(noticeTimerRef.current);
        noticeTimerRef.current = undefined;
      }
    };
  }, [notice, noticePath]);

  const filteredSessions = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return sessions.filter((item) => {
      if (methodFilter !== 'ALL' && item.method.toUpperCase() !== methodFilter) {
        return false;
      }
      if (!needle) {
        return true;
      }
      return [item.method, item.host, item.path, item.status, item.source, item.contentType, item.responsePreview, String(item.statusCode)]
        .filter(Boolean)
        .some((part) => part.toLowerCase().includes(needle));
    });
  }, [sessions, query, methodFilter]);

  const methodOptions = useMemo(() => {
    const common = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'CONNECT'];
    const seen = new Set(sessions.map((item) => item.method.toUpperCase()).filter(Boolean));
    return common.filter((method) => seen.has(method)).concat(
      Array.from(seen).filter((method) => !common.includes(method)).sort(),
    );
  }, [sessions]);

  const curlText = useMemo(() => detail ? buildCurl(detail) : '', [detail]);
  const rawRequestText = useMemo(() => detail ? buildRawRequest(detail) : '', [detail]);
  const rawResponseText = useMemo(() => detail ? buildRawResponse(detail) : '', [detail]);

  async function runAction(action: () => Promise<void> | void, success: string) {
    if (!hasWailsRuntime()) {
      showNotice('当前是浏览器预览，桌面运行时 API 未连接');
      return;
    }
    try {
      await action();
      showNotice(success);
      await refresh();
    } catch (error) {
      showError(error);
    }
  }

  function showNotice(message: string, path = '') {
    setNotice(message);
    setNoticePath(path);
  }

  function showError(error: unknown) {
    const message = error instanceof Error ? error.message : String(error);
    showNotice(message);
  }

  async function copyText(text: string, success: string) {
    if (!text) {
      showNotice('没有可复制内容');
      return;
    }
    try {
      if (hasWailsRuntime()) {
        await ClipboardSetText(text);
      } else {
        await navigator.clipboard?.writeText(text);
      }
      showNotice(success);
    } catch (error) {
      showError(error);
    }
  }

  async function exportHar() {
    if (!hasWailsRuntime()) {
      showNotice('当前是浏览器预览，桌面运行时 API 未连接');
      return;
    }
    try {
      const path = await ExportHAR();
      showNotice(`HAR 已导出：${path}`, path);
      await refresh();
    } catch (error) {
      showError(error);
    }
  }

  function selectSession(id: string) {
    selectedIdRef.current = id;
    setSelectedId(id);
  }

  function setUpstreamEditing(editing: boolean) {
    editingUpstreamRef.current = editing;
    setEditingUpstream(editing);
  }

  async function openNoticePath(path: string) {
    if (!path) {
      showNotice('文件路径为空');
      return;
    }
    try {
      await OpenPathLocation(path);
      showNotice('已打开导出文件位置');
    } catch (error) {
      showError(error);
    }
  }

  const primaryLanAddress = status.lanProxyAddresses[0] || '等待网卡地址';
  const upstreamLabel = status.upstreamProxy || '直连';

  return (
    <main className="shell">
      <header className="topbar">
        <div className="brand">
          <div className="brand-mark"><img src={appIcon} alt="老虎快跑"/></div>
          <div>
            <h1>老虎快跑</h1>
            <p>HTTP/HTTPS 抓包代理 · v{status.version}</p>
          </div>
        </div>
        <div className="top-actions">
          <button
            className={status.proxyRunning ? 'button danger' : 'button primary'}
            onClick={() => runAction(status.proxyRunning ? StopProxy : StartProxy, status.proxyRunning ? '代理已停止' : '代理已启动')}
            title={status.proxyRunning ? '停止代理服务' : '启动代理服务'}
          >
            {status.proxyRunning ? <Power size={16}/> : <Play size={16}/>}
            {status.proxyRunning ? '停止' : '启动'}
          </button>
          <button
            className={status.interceptHttps ? 'button active' : 'button'}
            onClick={() => runAction(() => SetHTTPSIntercept(!status.interceptHttps), status.interceptHttps ? 'HTTPS 解密已关闭' : 'HTTPS 解密已开启')}
            title="开启后会对 HTTPS CONNECT 执行本地 MITM 解密"
          >
            <ShieldCheck size={16}/>
            HTTPS
          </button>
          <button className="button" onClick={() => runAction(EnableSystemProxy, '已开启 Windows 系统代理')} title="把当前用户 Windows 系统代理指向老虎快跑">
            <Laptop size={16}/>
            系统代理
          </button>
          <button className="button" onClick={() => runAction(DisableSystemProxy, '已关闭 Windows 系统代理')} title="关闭当前用户 Windows 系统代理">
            <Ban size={16}/>
            关闭代理
          </button>
          <button className="button" onClick={exportHar} title="导出当前抓包记录">
            <FileDown size={16}/>
            HAR
          </button>
        </div>
      </header>

      <section className="status-strip">
        <Metric icon={<Circle size={11} fill={status.proxyRunning ? '#0f9f6e' : '#9ca3af'} strokeWidth={0}/>} label="代理状态" value={status.proxyRunning ? '运行中' : '已停止'}/>
        <Metric icon={<Globe2 size={16}/>} label="本机代理" value={status.localProxyAddress}/>
        <Metric icon={<Wifi size={16}/>} label="局域网代理" value={primaryLanAddress}/>
        <Metric icon={<Network size={16}/>} label="上游代理" value={upstreamLabel}/>
        <Metric icon={<KeyRound size={16}/>} label="证书安装" value={status.certificateUrl}/>
      </section>

      <section className="workspace">
        <aside className="sidebar">
          <div className="section-title">接入</div>
          <Endpoint icon={<Laptop size={18}/>} title="Windows / 浏览器" value={status.localProxyAddress} onCopy={() => runAction(CopyProxyAddress, '已复制本机代理地址')}/>
          <Endpoint icon={<Smartphone size={18}/>} title="iOS / Android Wi-Fi" value={primaryLanAddress} onCopy={() => copyText(primaryLanAddress, '已复制局域网代理地址')}/>
          <Endpoint icon={<Download size={18}/>} title="证书下载页" value={status.certificateUrl} onCopy={() => runAction(CopyCertificateURL, '已复制证书链接')}/>

          <div className="section-title space">上游代理</div>
          <div className="upstream-panel">
            <input
              value={upstreamDraft}
              onFocus={() => setUpstreamEditing(true)}
              onBlur={() => setUpstreamEditing(false)}
              onChange={(event) => setUpstreamDraft(event.target.value)}
              placeholder="http://127.0.0.1:7890"
            />
            <div className="upstream-actions">
              <button onClick={() => runAction(async () => {
                await SetUpstreamProxy(upstreamDraft);
                setUpstreamEditing(false);
              }, '上游代理已保存')}>
                应用
              </button>
              <button onClick={() => runAction(async () => {
                await ClearUpstreamProxy();
                setUpstreamDraft('');
                setUpstreamEditing(false);
              }, '上游代理已清除')}>
                直连
              </button>
            </div>
            <p className="hint">可填写本机或局域网 HTTP 代理地址，例如 <strong>http://127.0.0.1:7890</strong>。SOCKS 可填 <strong>socks5://127.0.0.1:7890</strong>。</p>
          </div>

          <div className="section-title space">证书</div>
          <button className="side-button" onClick={() => runAction(InstallRootCertificate, '根证书已安装到当前用户根证书库')}>
            <BadgeCheck size={16}/>
            安装到 Windows
          </button>
          <button className="side-button" onClick={() => runAction(OpenCertificateURL, '已打开证书下载页')}>
            <Globe2 size={16}/>
            打开证书页
          </button>
          <button className="side-button" onClick={() => runAction(OpenCertificateLocation, '已打开证书位置')}>
            <HardDrive size={16}/>
            证书文件位置
          </button>

          <div className="section-title space">模式</div>
          <div className="mode-row active-mode">
            <CheckCircle2 size={16}/>
            <span>HTTP/HTTPS 代理</span>
          </div>
          <div className="mode-row muted">
            <Eye size={16}/>
            <span>TUN/VPN 入口预留</span>
          </div>
          <p className="hint">{status.tunStatus}</p>
        </aside>

        <section className="traffic">
          <div className="traffic-toolbar">
            <div className="search">
              <Search size={16}/>
              <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索域名、路径、状态码、设备"/>
            </div>
            <div className="method-filter" aria-label="接口类型筛选">
              <span className="filter-label"><ListFilter size={14}/>接口</span>
              <button className={methodFilter === 'ALL' ? 'filter-chip active-filter' : 'filter-chip'} onClick={() => setMethodFilter('ALL')}>
                全部
              </button>
              {methodOptions.map((method) => (
                <button key={method} className={methodFilter === method ? 'filter-chip active-filter' : 'filter-chip'} onClick={() => setMethodFilter(method)}>
                  {method}
                </button>
              ))}
            </div>
            <button className="icon-button" onClick={() => refresh()} title="刷新">
              <RefreshCcw size={16}/>
            </button>
            <button className="icon-button" onClick={() => runAction(async () => {
              await ClearSessions();
              selectedIdRef.current = '';
              setSelectedId('');
              setDetail(null);
            }, '抓包列表已清空')} title="清空">
              <Trash2 size={16}/>
            </button>
          </div>

          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>方法</th>
                  <th>Host</th>
                  <th>Path</th>
                  <th>状态</th>
                  <th>耗时</th>
                  <th>大小</th>
                </tr>
              </thead>
              <tbody>
                {filteredSessions.length === 0 && (
                  <tr>
                    <td colSpan={6} className="empty">还没有流量。开启系统代理，或在手机 Wi-Fi 中填入局域网代理地址。</td>
                  </tr>
                )}
                {filteredSessions.map((item) => (
                  <tr key={item.id} className={selectedId === item.id ? 'selected' : ''} onClick={() => selectSession(item.id)}>
                    <td><Method value={item.method}/></td>
                    <td>
                      <div className="host-cell">{item.host || '-'}</div>
                      <div className="subtle">{item.source || '-'}</div>
                    </td>
                    <td className="path-cell">{item.path || item.url}</td>
                    <td><StatusCode code={item.statusCode} error={item.error}/></td>
                    <td>{item.durationMs}ms</td>
                    <td>{formatBytes(item.responseSize)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <aside className="detail">
          {detail ? (
            <>
              <div className="detail-head">
                <div>
                  <div className="detail-method"><Method value={detail.method}/></div>
                  <h2>{detail.host}</h2>
                  <p>{detail.path}</p>
                </div>
                <button className="icon-button" onClick={() => copyText(detail.url, '已复制 URL')} title="复制 URL">
                  <Copy size={16}/>
                </button>
              </div>
              <div className="quick-copy-bar">
                <button className="action-button" onClick={() => copyText(curlText, '已复制 cURL')} title="复制为可复现请求">
                  <Braces size={15}/>
                  cURL
                </button>
                <button className="action-button" onClick={() => copyText(bodyTextForCopy(detail.requestBody), bodyCopyMessage('请求体', detail.requestBody))} title="复制请求体">
                  <Copy size={15}/>
                  请求体
                </button>
                <button className="action-button" onClick={() => copyText(bodyTextForCopy(detail.responseBody), bodyCopyMessage('响应体', detail.responseBody))} title="复制响应体">
                  <Copy size={15}/>
                  响应体
                </button>
                <button className="action-button" onClick={() => copyText(rawRequestText, '已复制完整请求')} title="复制完整请求报文">
                  <Copy size={15}/>
                  完整请求
                </button>
                <button className="action-button" onClick={() => copyText(rawResponseText, '已复制完整响应')} title="复制完整响应报文">
                  <Copy size={15}/>
                  完整响应
                </button>
                <button className="action-button" onClick={() => copyText(detail.url, '已复制 URL')} title="复制 URL">
                  <Copy size={15}/>
                  URL
                </button>
              </div>

              <div className="tabs">
                <button className={detailTab === 'overview' ? 'tab active-tab' : 'tab'} onClick={() => setDetailTab('overview')}>概览</button>
                <button className={detailTab === 'request' ? 'tab active-tab' : 'tab'} onClick={() => setDetailTab('request')}>请求</button>
                <button className={detailTab === 'response' ? 'tab active-tab' : 'tab'} onClick={() => setDetailTab('response')}>响应</button>
                <button className={detailTab === 'cert' ? 'tab active-tab' : 'tab'} onClick={() => setDetailTab('cert')}>证书</button>
              </div>

              {detailTab === 'overview' && <Overview detail={detail}/>}
              {detailTab === 'request' && <BodyPanel title="Request" headers={detail.requestHeaders} body={detail.requestBody} rawText={rawRequestText} curlText={curlText} onCopyText={copyText}/>}
              {detailTab === 'response' && <BodyPanel title="Response" headers={detail.responseHeaders} body={detail.responseBody} rawText={rawResponseText} onCopyText={copyText}/>}
              {detailTab === 'cert' && <CertificatePanel detail={detail} status={status}/>}
            </>
          ) : (
            <div className="empty-detail">
              <Braces size={24}/>
              <p>选择一条请求查看详情</p>
            </div>
          )}
        </aside>
      </section>

      {notice && (
        <div className="toast">
          <button className="toast-close" onClick={() => showNotice('')} title="关闭">×</button>
          <div className="toast-message">{notice}</div>
          {noticePath && (
            <div className="toast-actions">
              <button onClick={() => openNoticePath(noticePath)}>
                <HardDrive size={14}/>
                打开位置
              </button>
              <button onClick={() => copyText(noticePath, '已复制 HAR 文件路径')}>
                <Copy size={14}/>
                复制路径
              </button>
            </div>
          )}
        </div>
      )}
    </main>
  );
}

function hasWailsRuntime(): boolean {
  return Boolean(window.go?.main?.App);
}

function Metric({icon, label, value}: {icon: ReactNode; label: string; value: string}) {
  return (
    <div className="metric">
      {icon}
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function Endpoint({icon, title, value, onCopy}: {icon: ReactNode; title: string; value: string; onCopy: () => void}) {
  return (
    <div className="endpoint">
      <div className="endpoint-icon">{icon}</div>
      <div>
        <span>{title}</span>
        <strong>{value}</strong>
      </div>
      <button className="mini-button" onClick={onCopy} title="复制">
        <Copy size={14}/>
      </button>
    </div>
  );
}

function Method({value}: {value: string}) {
  return <span className={`method method-${value.toLowerCase()}`}>{value}</span>;
}

function StatusCode({code, error}: {code: number; error: string}) {
  const label = error ? 'ERR' : code || '-';
  const tone = error ? 'bad' : code >= 500 ? 'bad' : code >= 400 ? 'warn' : code >= 200 ? 'good' : 'plain';
  return <span className={`status-code ${tone}`}>{label}</span>;
}

function Overview({detail}: {detail: SessionDetail}) {
  return (
    <div className="detail-content">
      <InfoGrid rows={[
        ['URL', detail.url],
        ['协议', detail.protocol || '-'],
        ['来源', detail.source || '-'],
        ['HTTPS 解密', detail.interceptedTls ? '已解密' : detail.tunnelOnly ? '隧道模式' : '-'],
        ['Content-Type', detail.contentType || '-'],
        ['耗时', `${detail.durationMs}ms`],
        ['请求大小', formatBytes(detail.requestSize)],
        ['响应大小', formatBytes(detail.responseSize)],
      ]}/>
      {detail.error && <pre className="code error-block">{detail.error}</pre>}
      {detail.responsePreview && <pre className="code preview">{detail.responsePreview}</pre>}
    </div>
  );
}

function BodyPanel({title, headers, body, rawText, curlText, onCopyText}: {title: string; headers: Record<string, string[]>; body: BodyView; rawText?: string; curlText?: string; onCopyText: (text: string, success: string) => void | Promise<void>}) {
  const bodyText = bodyTextForCopy(body);
  return (
    <div className="detail-content">
      <div className="panel-title-row">
        <h3>{title} Headers</h3>
        <div className="panel-actions">
          {curlText && <button onClick={() => onCopyText(curlText, '已复制 cURL')}>复制 cURL</button>}
          {rawText && <button onClick={() => onCopyText(rawText, `${title} 完整报文已复制`)}>复制完整</button>}
          <button onClick={() => onCopyText(formatHeaders(headers), `${title} Headers 已复制`)}>复制 Headers</button>
        </div>
      </div>
      <pre className="code">{formatHeaders(headers)}</pre>
      <div className="panel-title-row">
        <h3>{title} Body</h3>
        <div className="panel-actions">
          <button onClick={() => onCopyText(bodyText, bodyCopyMessage(`${title} Body`, body))}>复制 Body</button>
        </div>
      </div>
      <pre className="code body-code">{body.text || body.base64 || '(empty)'}</pre>
      <div className="body-meta">{formatBytes(body.size)} · {body.contentType || 'unknown'} {body.truncated ? '· preview truncated' : ''}</div>
    </div>
  );
}

function CertificatePanel({detail, status}: {detail: SessionDetail; status: Status}) {
  return (
    <div className="detail-content">
      <InfoGrid rows={[
        ['根证书', status.certificateSubject],
        ['证书文件', status.certificatePath],
        ['过期日期', status.certificateExpiresAt],
        ['当前请求', detail.interceptedTls ? '使用老虎快跑动态证书解密' : '未进行 HTTPS 解密'],
        ['Server Name', detail.certificate.serverName || detail.host],
        ['Issuer', detail.certificate.issuer || '-'],
      ]}/>
      <p className="hint">App 做了 SSL Pinning 或不信任用户 CA 时，VPN/TUN 也只能看到连接，无法看到 HTTPS 明文。</p>
    </div>
  );
}

function InfoGrid({rows}: {rows: [string, string][]}) {
  return (
    <div className="info-grid">
      {rows.map(([label, value]) => (
        <div key={label}>
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  );
}

function formatHeaders(headers: Record<string, string[]>): string {
  const keys = Object.keys(headers || {}).sort();
  if (keys.length === 0) {
    return '(empty)';
  }
  return keys.flatMap((key) => headers[key].map((value) => `${key}: ${value}`)).join('\n');
}

function bodyTextForCopy(body: BodyView): string {
  if (!body) {
    return '';
  }
  if (body.text && body.text !== '[binary body]') {
    return body.text;
  }
  return body.base64 || body.text || '';
}

function bodyCopyMessage(label: string, body: BodyView): string {
  if (body.truncated) {
    return `${label} 预览已复制，原始内容超过 1MB 已截断`;
  }
  if (body.base64 && body.text === '[binary body]') {
    return `${label} 已复制为 Base64`;
  }
  return `${label} 已复制`;
}

function buildCurl(detail: SessionDetail): string {
  const method = (detail.method || 'GET').toUpperCase();
  const parts = ['curl', '--location', shellQuote(requestUrl(detail)), '--request', shellQuote(method)];
  const ignoredHeaders = new Set(['accept-encoding', 'connection', 'content-length', 'host', 'proxy-connection']);

  Object.keys(detail.requestHeaders || {}).sort().forEach((key) => {
    if (ignoredHeaders.has(key.toLowerCase())) {
      return;
    }
    detail.requestHeaders[key].forEach((value) => {
      parts.push('--header', shellQuote(`${key}: ${value}`));
    });
  });

  const body = bodyTextForCopy(detail.requestBody);
  if (body && body !== '[binary body]') {
    parts.push('--data-raw', shellQuote(body));
  }
  return parts.join(' ');
}

function buildRawRequest(detail: SessionDetail): string {
  const method = (detail.method || 'GET').toUpperCase();
  const path = requestPath(detail);
  const protocol = detail.protocol && detail.protocol.startsWith('HTTP/') ? detail.protocol : 'HTTP/1.1';
  const lines = [`${method} ${path} ${protocol}`];
  const headers = detail.requestHeaders || {};
  const hasHost = Object.keys(headers).some((key) => key.toLowerCase() === 'host');

  if (detail.host && !hasHost) {
    lines.push(`Host: ${detail.host}`);
  }
  appendHeaderLines(lines, headers);

  const body = bodyTextForCopy(detail.requestBody);
  return body ? `${lines.join('\r\n')}\r\n\r\n${body}` : `${lines.join('\r\n')}\r\n\r\n`;
}

function buildRawResponse(detail: SessionDetail): string {
  const protocol = detail.protocol && detail.protocol.startsWith('HTTP/') ? detail.protocol : 'HTTP/1.1';
  const statusCode = detail.statusCode || 0;
  const statusText = detail.status || statusLabel(statusCode);
  const statusLine = statusCode ? `${protocol} ${statusCode} ${statusText}`.trim() : `${protocol} ${statusText || '0'}`;
  const lines = [statusLine];

  appendHeaderLines(lines, detail.responseHeaders || {});
  const body = bodyTextForCopy(detail.responseBody);
  return body ? `${lines.join('\r\n')}\r\n\r\n${body}` : `${lines.join('\r\n')}\r\n\r\n`;
}

function appendHeaderLines(lines: string[], headers: Record<string, string[]>) {
  Object.keys(headers || {}).sort().forEach((key) => {
    headers[key].forEach((value) => lines.push(`${key}: ${value}`));
  });
}

function requestPath(detail: SessionDetail): string {
  if (detail.path) {
    return detail.path;
  }
  try {
    const parsed = new URL(requestUrl(detail));
    return `${parsed.pathname}${parsed.search}` || '/';
  } catch {
    return '/';
  }
}

function statusLabel(code: number): string {
  const labels: Record<number, string> = {
    200: 'OK',
    201: 'Created',
    202: 'Accepted',
    204: 'No Content',
    301: 'Moved Permanently',
    302: 'Found',
    304: 'Not Modified',
    400: 'Bad Request',
    401: 'Unauthorized',
    403: 'Forbidden',
    404: 'Not Found',
    500: 'Internal Server Error',
    502: 'Bad Gateway',
    503: 'Service Unavailable',
  };
  return labels[code] || '';
}

function requestUrl(detail: SessionDetail): string {
  if (detail.url) {
    return detail.url;
  }
  const host = detail.host || '';
  const path = detail.path || '/';
  if (!host) {
    return path;
  }
  const scheme = detail.scheme || (detail.interceptedTls ? 'https' : 'http');
  return `${scheme}://${host}${path.startsWith('/') ? path : `/${path}`}`;
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

function formatBytes(value: number): string {
  if (!value) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

export default App;
