const DEFAULT_SETTINGS = {
  host: '127.0.0.1',
  port: 8080,
  enabled: false,
  bypassList: ['<local>'],
};

chrome.runtime.onInstalled.addListener(async () => {
  const current = await chrome.storage.local.get(DEFAULT_SETTINGS);
  await chrome.storage.local.set({...DEFAULT_SETTINGS, ...current});
  await refreshBadge();
});

chrome.runtime.onStartup.addListener(refreshBadge);

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  handleMessage(message).then(sendResponse).catch((error) => {
    sendResponse({ok: false, error: error.message || String(error)});
  });
  return true;
});

async function handleMessage(message) {
  switch (message?.type) {
    case 'get-state':
      return {ok: true, state: await getState()};
    case 'enable-proxy':
      return {ok: true, state: await enableProxy(message.host, message.port)};
    case 'disable-proxy':
      return {ok: true, state: await disableProxy()};
    case 'open-cert':
      return {ok: true, state: await openCertPage()};
    default:
      throw new Error('Unknown message');
  }
}

async function getState() {
  const settings = await chrome.storage.local.get(DEFAULT_SETTINGS);
  const proxyConfig = await chrome.proxy.settings.get({incognito: false});
  const health = await checkTigerRun(settings.host, settings.port);
  const enabled = isTigerRunProxy(proxyConfig.value, settings.host, settings.port);

  if (settings.enabled !== enabled) {
    await chrome.storage.local.set({enabled});
  }
  await setBadge(enabled);

  return {
    ...settings,
    enabled,
    levelOfControl: proxyConfig.levelOfControl,
    health,
  };
}

async function enableProxy(host, port) {
  const normalizedHost = normalizeHost(host);
  const normalizedPort = normalizePort(port);
  const bypassList = DEFAULT_SETTINGS.bypassList;

  const config = {
    mode: 'fixed_servers',
    rules: {
      singleProxy: {
        scheme: 'http',
        host: normalizedHost,
        port: normalizedPort,
      },
      bypassList,
    },
  };

  await chrome.proxy.settings.set({value: config, scope: 'regular'});
  await chrome.storage.local.set({
    host: normalizedHost,
    port: normalizedPort,
    enabled: true,
    bypassList,
  });
  await setBadge(true);
  return getState();
}

async function disableProxy() {
  await chrome.proxy.settings.clear({scope: 'regular'});
  await chrome.storage.local.set({enabled: false});
  await setBadge(false);
  return getState();
}

async function openCertPage() {
  const settings = await chrome.storage.local.get(DEFAULT_SETTINGS);
  await chrome.tabs.create({url: `http://${settings.host}:${settings.port}/cert`});
  return getState();
}

async function checkTigerRun(host, port) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 1200);
  try {
    const response = await fetch(`http://${host}:${port}/__laohukuaipao/health`, {
      cache: 'no-store',
      signal: controller.signal,
    });
    if (!response.ok) {
      return {ok: false, message: `HTTP ${response.status}`};
    }
    return await response.json();
  } catch (error) {
    return {
      ok: false,
      message: error.name === 'AbortError' ? '老虎快跑没有响应' : '无法连接老虎快跑',
    };
  } finally {
    clearTimeout(timeout);
  }
}

async function refreshBadge() {
  const settings = await chrome.storage.local.get(DEFAULT_SETTINGS);
  const proxyConfig = await chrome.proxy.settings.get({incognito: false});
  await setBadge(isTigerRunProxy(proxyConfig.value, settings.host, settings.port));
}

async function setBadge(enabled) {
  await chrome.action.setBadgeText({text: enabled ? 'ON' : ''});
  await chrome.action.setBadgeBackgroundColor({color: enabled ? '#0f766e' : '#6b7280'});
}

function isTigerRunProxy(config, host, port) {
  const proxy = config?.rules?.singleProxy || config?.rules?.proxyForHttp || config?.rules?.proxyForHttps;
  return config?.mode === 'fixed_servers' &&
    proxy?.scheme === 'http' &&
    proxy?.host === host &&
    Number(proxy?.port) === Number(port);
}

function normalizeHost(host) {
  const value = String(host || '').trim();
  if (!value) {
    throw new Error('代理地址不能为空');
  }
  if (!/^[a-zA-Z0-9.\-:]+$/.test(value)) {
    throw new Error('代理地址格式不正确');
  }
  return value;
}

function normalizePort(port) {
  const value = Number(port);
  if (!Number.isInteger(value) || value < 1 || value > 65535) {
    throw new Error('端口必须在 1 到 65535 之间');
  }
  return value;
}
