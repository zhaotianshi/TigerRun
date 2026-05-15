const hostInput = document.querySelector('#host');
const portInput = document.querySelector('#port');
const statusDot = document.querySelector('#statusDot');
const statusTitle = document.querySelector('#statusTitle');
const statusText = document.querySelector('#statusText');
const enableBtn = document.querySelector('#enableBtn');
const disableBtn = document.querySelector('#disableBtn');
const certBtn = document.querySelector('#certBtn');
const copyBtn = document.querySelector('#copyBtn');

document.addEventListener('DOMContentLoaded', refresh);
enableBtn.addEventListener('click', enableProxy);
disableBtn.addEventListener('click', disableProxy);
certBtn.addEventListener('click', openCertPage);
copyBtn.addEventListener('click', copyProxyAddress);

async function refresh() {
  const response = await send({type: 'get-state'});
  if (!response.ok) {
    renderError(response.error);
    return;
  }
  renderState(response.state);
}

async function enableProxy() {
  setBusy(true);
  const response = await send({
    type: 'enable-proxy',
    host: hostInput.value,
    port: portInput.value,
  });
  setBusy(false);
  if (!response.ok) {
    renderError(response.error);
    return;
  }
  renderState(response.state);
}

async function disableProxy() {
  setBusy(true);
  const response = await send({type: 'disable-proxy'});
  setBusy(false);
  if (!response.ok) {
    renderError(response.error);
    return;
  }
  renderState(response.state);
}

async function openCertPage() {
  await send({type: 'open-cert'});
  window.close();
}

async function copyProxyAddress() {
  const address = `${hostInput.value.trim()}:${portInput.value.trim()}`;
  await navigator.clipboard.writeText(address);
  statusTitle.textContent = '已复制代理地址';
  statusText.textContent = address;
}

function renderState(state) {
  hostInput.value = state.host || '127.0.0.1';
  portInput.value = state.port || 8080;
  const controllable = state.levelOfControl === 'controllable_by_this_extension' ||
    state.levelOfControl === 'controlled_by_this_extension';

  enableBtn.disabled = !controllable;
  disableBtn.disabled = !controllable;

  statusDot.className = 'dot';
  if (!controllable) {
    statusDot.classList.add('warn');
    statusTitle.textContent = '代理设置被其他扩展控制';
    statusText.textContent = state.levelOfControl || '不可控制';
    return;
  }

  if (state.enabled && state.health?.ok) {
    statusDot.classList.add('on');
    statusTitle.textContent = 'Chrome 代理已开启';
    statusText.textContent = `正在转发到 ${state.host}:${state.port}`;
    return;
  }

  if (state.enabled && !state.health?.ok) {
    statusDot.classList.add('warn');
    statusTitle.textContent = '代理已开启，但后台未响应';
    statusText.textContent = state.health?.message || '请先启动老虎快跑桌面程序';
    return;
  }

  statusDot.classList.add(state.health?.ok ? 'warn' : 'off');
  statusTitle.textContent = state.health?.ok ? '老虎快跑在线' : '老虎快跑未连接';
  statusText.textContent = state.health?.ok ? '点击开启 Chrome 代理开始抓包' : '请先启动桌面程序';
}

function renderError(message) {
  statusDot.className = 'dot off';
  statusTitle.textContent = '操作失败';
  statusText.textContent = message || '未知错误';
}

function setBusy(busy) {
  enableBtn.disabled = busy;
  disableBtn.disabled = busy;
  enableBtn.textContent = busy ? '处理中...' : '开启 Chrome 代理';
}

function send(message) {
  return chrome.runtime.sendMessage(message);
}
