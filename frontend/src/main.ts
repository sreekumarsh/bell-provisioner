import './app.css';
import {
  GetConfig,
  SaveConfig,
  Login,
  DiscoverDevices,
  Provision,
  Install,
  Verify,
  TestSSH,
} from '../wailsjs/go/main/App';
import { config } from '../wailsjs/go/models';
import type { main } from '../wailsjs/go/models';

type Step = 1 | 2 | 3 | 4 | 5;

const stepLabels = ['Environment', 'Login', 'Discover', 'Provision', 'Install'];

let currentStep: Step = 1;
let selectedHost = '';
let provisionResult: main.ProvisionResult | null = null;
let savedCfg: config.AppConfig | null = null;

const state = {
  gatewayURL: 'https://api.vyooham.com',
  backendProfile: 'vps',
  sshUser: 'pi',
  sshHost: 'raspberrypi.local',
  sshPort: 22,
  macIP: '',
  phone: '',
  password: '',
  sshPassword: '',
  serial: '',
  hwVersion: '1.0',
  deployAgentEnv: true,
  checkoutAgent: true,
  githubToken: '',
  overwriteSerial: true,
  manualHost: '',
};

async function init() {
  try {
    const cfg = await GetConfig();
    savedCfg = cfg;
    state.gatewayURL = cfg.gateway_url || state.gatewayURL;
    state.backendProfile = cfg.backend_profile || 'vps';
    state.sshUser = (cfg.ssh_user || state.sshUser || 'sreekumar').toLowerCase();
    state.sshHost = cfg.ssh_host || 'raspberrypi.local';
    state.sshPort = cfg.ssh_port || 22;
    state.macIP = cfg.mac_ip || '';
    state.phone = cfg.phone || '';
    state.githubToken = cfg.github_token || '';
    state.sshPassword = cfg.ssh_password || '';
  } catch (e) {
    console.error(e);
  }
  render();
}

function render() {
  const app = document.getElementById('app')!;
  app.innerHTML = `
    <h1>Bell Provisioner</h1>
    <p class="subtitle">Factory / lab device provisioning for VPS or Mac LAN</p>
    ${renderSteps()}
    <div class="panel">${renderPanel()}</div>
  `;
  bindEvents();
}

function renderSteps(): string {
  return `<div class="steps">${stepLabels.map((label, i) => {
    const n = (i + 1) as Step;
    let cls = 'step-indicator';
    if (n === currentStep) cls += ' active';
    else if (n < currentStep) cls += ' done';
    return `<div class="${cls}">${n}. ${label}</div>`;
  }).join('')}</div>`;
}

function renderPanel(): string {
  switch (currentStep) {
    case 1: return renderEnv();
    case 2: return renderLogin();
    case 3: return renderDiscover();
    case 4: return renderProvision();
    case 5: return renderInstall();
    default: return '';
  }
}

function renderEnv(): string {
  return `
    <h2>Environment</h2>
    <div class="field">
      <label>Gateway URL</label>
      <input id="gatewayURL" value="${esc(state.gatewayURL)}" placeholder="https://api.vyooham.com" />
    </div>
    <div class="field">
      <label>Target backend</label>
      <select id="backendProfile">
        <option value="vps" ${state.backendProfile === 'vps' ? 'selected' : ''}>VPS (api.vyooham.com)</option>
        <option value="mac" ${state.backendProfile === 'mac' ? 'selected' : ''}>Mac LAN dev</option>
      </select>
    </div>
    <div class="field ${state.backendProfile === 'mac' ? '' : 'hidden'}" id="macIPField">
      <label>Mac IP (for agent.env)</label>
      <input id="macIP" value="${esc(state.macIP)}" placeholder="192.168.4.66" />
    </div>
    <div class="row">
      <div class="field">
        <label>SSH user (Pi Linux account)</label>
        <input id="sshUser" value="${esc(state.sshUser)}" placeholder="sreekumar" />
        <p style="color:var(--muted);font-size:0.85rem;margin:4px 0 0">Pi login from Raspberry Pi Imager (lowercase). Not your Mac username.</p>
      </div>
      <div class="field">
        <label>SSH host</label>
        <input id="sshHost" value="${esc(state.sshHost)}" />
      </div>
      <div class="field" style="max-width:80px">
        <label>Port</label>
        <input id="sshPort" type="number" value="${state.sshPort}" />
      </div>
    </div>
    <div class="field">
      <label>SSH password</label>
      <input id="sshPassword" type="password" autocomplete="off" />
      <p style="color:var(--muted);font-size:0.85rem;margin:8px 0 0">
        Password auth only (Mac SSH keys are not used). Saved to local config when you continue.
      </p>
      ${state.sshPassword ? '<p style="color:var(--success);font-size:0.85rem;margin:4px 0 0">SSH password saved in config</p>' : ''}
    </div>
    <div class="field">
      <label>GitHub token (read-only)</label>
      <input id="githubToken" type="password" autocomplete="off" placeholder="${state.githubToken ? 'leave blank to keep saved token' : 'ghp_… or github_pat_…'}" />
      ${state.githubToken ? '<p style="color:var(--success);font-size:0.85rem;margin:4px 0 0">GitHub token saved in config</p>' : ''}
      <p style="color:var(--muted);font-size:0.85rem;margin:8px 0 0">
        Downloads the latest <code>doorbell-agent-linux-arm64</code> artifact. Saved to local config when you continue (not in git).
      </p>
    </div>
    <div class="actions">
      <button class="primary" id="btnNextEnv">Continue</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderLogin(): string {
  return `
    <h2>Admin login</h2>
    <div class="field">
      <label>Phone (+country code)</label>
      <input id="phone" value="${esc(state.phone)}" placeholder="+919876543210" />
    </div>
    <div class="field">
      <label>Password</label>
      <input id="password" type="password" />
    </div>
    <div class="actions">
      <button class="secondary" id="btnBack">Back</button>
      <button class="primary" id="btnLogin">Login</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderDiscover(): string {
  const host = selectedHost || state.sshHost;
  return `
    <h2>Discover device</h2>
    <p style="color:var(--muted);font-size:0.85rem;margin:0 0 16px">
      SSH host from Environment is pre-selected (<strong>${esc(host)}</strong>). Use <em>Continue</em> directly, or scan the LAN to find other Pis.
    </p>
    <div class="field">
      <label>Extra host to probe (optional)</label>
      <input id="manualHost" value="${esc(state.manualHost)}" placeholder="192.168.1.42" />
    </div>
    <button class="secondary" id="btnScan">Scan full network (slow)</button>
    <button class="secondary" id="btnProbeHost">Probe configured host</button>
    <ul class="device-list" id="deviceList"></ul>
    <div class="field">
      <label>Selected host (used for install)</label>
      <input id="selectedHost" value="${esc(host)}" />
    </div>
    <div class="actions">
      <button class="secondary" id="btnBack">Back</button>
      <button class="secondary" id="btnTestSSH">Test SSH</button>
      <button class="primary" id="btnNextDiscover">Continue</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderProvision(): string {
  return `
    <h2>Provision in cloud</h2>
    <div class="field">
      <label>Serial number (from device label)</label>
      <input id="serial" value="${esc(state.serial)}" placeholder="DB-2605-0001" />
    </div>
    <div class="field">
      <label>Hardware version</label>
      <input id="hwVersion" value="${esc(state.hwVersion)}" />
    </div>
    <label class="checkbox-field">
      <input type="checkbox" id="overwriteSerial" ${state.overwriteSerial ? 'checked' : ''} />
      Overwrite if serial already provisioned (re-provision with new keys)
    </label>
    <div class="actions">
      <button class="secondary" id="btnBack">Back</button>
      <button class="primary" id="btnProvision">Register device</button>
    </div>
    <div id="provisionSummary"></div>
    <div id="msg"></div>
  `;
}

function renderInstall(): string {
  const summary = provisionResult ? `
    <div class="summary">
      <strong>Device ID:</strong> <code>${esc(provisionResult.device_id)}</code><br/>
      <strong>Serial:</strong> ${esc(provisionResult.serial_number)}<br/>
      <strong>MQTT user:</strong> <code>${esc(provisionResult.mqtt_username)}</code>
    </div>
  ` : '';

  return `
    <h2>Install on device</h2>
    ${summary}
    <div class="field" style="margin-top:16px">
      <label>SSH password override (optional)</label>
      <input id="sshPasswordOverride" type="password" autocomplete="off" placeholder="${state.sshPassword ? 'Using Environment password' : 'Required if not set on Environment'}" />
    </div>
    <label class="checkbox-field">
      <input type="checkbox" id="checkoutAgent" ${state.checkoutAgent ? 'checked' : ''} />
      Download latest CI agent build and install on Pi
    </label>
    <label class="checkbox-field">
      <input type="checkbox" id="deployAgentEnv" ${state.deployAgentEnv ? 'checked' : ''} />
      Deploy agent.env for ${state.backendProfile === 'mac' ? 'Mac LAN' : 'VPS'}
    </label>
    <p class="hint" style="margin-top:8px;color:var(--muted);font-size:13px">
      Requires a read-only GitHub token on Environment (Actions read + private repo access).
    </p>
    <div class="actions">
      <button class="secondary" id="btnBack">Back</button>
      <button class="primary" id="btnInstall">Install on Pi</button>
      <button class="primary" id="btnVerify">Verify MQTT</button>
    </div>
    <div id="verifyResult"></div>
    <div id="msg"></div>
  `;
}

function bindEvents() {
  document.getElementById('btnBack')?.addEventListener('click', () => {
    if (currentStep > 1) {
      currentStep = (currentStep - 1) as Step;
      render();
    }
  });

  document.getElementById('btnNextEnv')?.addEventListener('click', async () => {
    readEnvFields();
    const pw = sshPassword();
    if (!pw) {
      setMsg('SSH password is required', true);
      return;
    }
    state.sshPassword = pw;
    const gh = githubToken();
    if (!gh) {
      setMsg('GitHub token is required to download the agent CI artifact', true);
      return;
    }
    state.githubToken = gh;
    const cfg = buildAppConfig({ github_token: gh, ssh_password: pw });
    await SaveConfig(cfg);
    savedCfg = cfg;
    currentStep = 2;
    render();
  });

  document.getElementById('backendProfile')?.addEventListener('change', (e) => {
    state.backendProfile = (e.target as HTMLSelectElement).value;
    document.getElementById('macIPField')?.classList.toggle('hidden', state.backendProfile !== 'mac');
  });

  document.getElementById('btnLogin')?.addEventListener('click', async () => {
    state.phone = val('phone');
    state.password = val('password');
    setMsg('Logging in…');
    try {
      const result = await Login(state.phone, state.password);
      setMsg(`Logged in as ${result.name || result.phone} (${result.role})`, false);
      currentStep = 3;
      initDiscoverStep();
      setTimeout(render, 600);
    } catch (e: any) {
      setMsg(e?.message || String(e), true);
    }
  });

  document.getElementById('btnProbeHost')?.addEventListener('click', () => {
    void runDiscover(false);
  });

  document.getElementById('btnScan')?.addEventListener('click', () => {
    void runDiscover(true);
  });

  document.getElementById('btnTestSSH')?.addEventListener('click', async () => {
    selectedHost = val('selectedHost') || selectedHost;
    const pw = sshPassword();
    if (!pw) {
      setMsg('Set SSH password on the Environment step first', true);
      return;
    }
    setMsg('Testing SSH…');
    try {
      await TestSSH(selectedHost, state.sshUser, state.sshPort, pw);
      setMsg('SSH connection OK', false);
    } catch (e: any) {
      setMsg(e?.message || String(e), true);
    }
  });

  document.getElementById('btnNextDiscover')?.addEventListener('click', () => {
    selectedHost = val('selectedHost') || selectedHost || state.sshHost;
    if (!selectedHost) {
      setMsg('Enter a device host (or set SSH host on Environment)', true);
      return;
    }
    state.sshHost = selectedHost;
    currentStep = 4;
    render();
  });

  document.getElementById('btnProvision')?.addEventListener('click', async () => {
    state.serial = val('serial');
    state.hwVersion = val('hwVersion') || '1.0';
    state.overwriteSerial = (document.getElementById('overwriteSerial') as HTMLInputElement)?.checked ?? true;
    setMsg('Provisioning…');
    try {
      provisionResult = await Provision({
        serial: state.serial,
        hw_version: state.hwVersion,
        overwrite: state.overwriteSerial,
      });
      document.getElementById('provisionSummary')!.innerHTML = `
        <div class="success summary" style="margin-top:16px">
          Registered <strong>${esc(provisionResult!.device_id)}</strong><br/>
          MQTT password shown once — will be written to the Pi on install.
        </div>`;
      setMsg('');
      currentStep = 5;
      setTimeout(render, 800);
    } catch (e: any) {
      setMsg(e?.message || String(e), true);
    }
  });

  document.getElementById('btnInstall')?.addEventListener('click', async () => {
    const pw = sshPassword();
    if (!pw) {
      setMsg('SSH password is required (Environment step or override below)', true);
      return;
    }
    const gh = githubToken();
    if (state.checkoutAgent && !gh) {
      setMsg('GitHub token is required (Environment step)', true);
      return;
    }
    state.sshPassword = pw;
    state.deployAgentEnv = (document.getElementById('deployAgentEnv') as HTMLInputElement)?.checked ?? true;
    state.checkoutAgent = (document.getElementById('checkoutAgent') as HTMLInputElement)?.checked ?? true;
    setMsg('Installing via SSH…');
    try {
      const result = await Install({
        host: selectedHost || state.sshHost,
        ssh_user: state.sshUser,
        ssh_port: state.sshPort,
        ssh_password: pw,
        deploy_agent_env: state.deployAgentEnv,
        checkout_agent: state.checkoutAgent,
        github_token: githubToken(),
      });
      setMsg(result.message + (result.agent_active ? ' · Agent active' : ''), !result.agent_active);
    } catch (e: any) {
      setMsg(e?.message || String(e), true);
    }
  });

  document.getElementById('btnVerify')?.addEventListener('click', async () => {
    setMsg('Verifying MQTT…');
    try {
      const result = await Verify({
        device_id: provisionResult?.device_id || '',
        mqtt_username: provisionResult?.mqtt_username || '',
        mqtt_password: provisionResult?.mqtt_password || '',
      });
      document.getElementById('verifyResult')!.innerHTML = `
        <div class="${result.mqtt.connected ? 'success' : 'error'}" style="margin-top:12px">
          ${esc(result.mqtt.message)}
        </div>`;
      setMsg('');
    } catch (e: any) {
      setMsg(e?.message || String(e), true);
    }
  });
}

function initDiscoverStep() {
  selectedHost = state.sshHost;
  if (!state.manualHost) state.manualHost = state.sshHost;
}

async function runDiscover(fullLAN: boolean) {
  state.manualHost = val('manualHost');
  const list = document.getElementById('deviceList');
  if (!list) return;
  list.innerHTML = fullLAN
    ? '<li class="spinner">Scanning LAN… (may take up to 45s)</li>'
    : '<li class="spinner">Probing configured host…</li>';
  setMsg('');
  try {
    const devices = (await DiscoverDevices(state.manualHost, fullLAN)) ?? [];
    if (devices.length === 0) {
      list.innerHTML = '<li style="cursor:default;color:var(--muted)">No SSH/setup on probed hosts — check IP or use Selected host below</li>';
      return;
    }
    list.innerHTML = devices.map(d => `
      <li data-host="${esc(d.host)}" data-serial="${esc(d.serial_number || '')}">
        <strong>${esc(d.host)}</strong>
        <div class="meta">
          SSH: ${d.ssh_reachable ? '✓' : '✗'}
          · Setup :4444: ${d.setup_server ? '✓' : '✗'}
          ${d.serial_number ? ` · Serial: ${esc(d.serial_number)}` : ''}
        </div>
      </li>
    `).join('');
    list.querySelectorAll('li[data-host]').forEach(li => {
      li.addEventListener('click', () => {
        selectedHost = li.getAttribute('data-host') || '';
        const serial = li.getAttribute('data-serial');
        if (serial) state.serial = serial;
        const input = document.getElementById('selectedHost') as HTMLInputElement;
        if (input) input.value = selectedHost;
        list.querySelectorAll('li').forEach(el => el.classList.remove('selected'));
        li.classList.add('selected');
      });
    });
  } catch (e: any) {
    list.innerHTML = '';
    setMsg(e?.message || String(e), true);
  }
}

function buildAppConfig(overrides: Partial<config.AppConfig> = {}): config.AppConfig {
  return config.AppConfig.createFrom({
    gateway_url: state.gatewayURL,
    backend_profile: state.backendProfile,
    ssh_user: state.sshUser,
    ssh_host: state.sshHost,
    ssh_port: state.sshPort,
    mac_ip: state.macIP,
    phone: state.phone,
    github_token: state.githubToken,
    ssh_password: state.sshPassword,
    agent_repo_url: savedCfg?.agent_repo_url || 'git@github.com:sreekumarsh/pi-streamer.git',
    agent_repo_branch: savedCfg?.agent_repo_branch || 'main',
    agent_repo_path: savedCfg?.agent_repo_path || '',
    agent_artifact_name: savedCfg?.agent_artifact_name || 'doorbell-agent-linux-arm64',
    ...overrides,
  });
}

function readEnvFields() {
  state.gatewayURL = val('gatewayURL');
  state.backendProfile = (document.getElementById('backendProfile') as HTMLSelectElement)?.value || 'vps';
  state.sshUser = val('sshUser').toLowerCase();
  state.sshHost = val('sshHost');
  state.sshPort = parseInt(val('sshPort') || '22', 10);
  state.macIP = val('macIP');
  const envPw = val('sshPassword');
  if (envPw) state.sshPassword = envPw;
  const gh = val('githubToken');
  if (isNewGitHubToken(gh)) state.githubToken = gh;
}

function isNewGitHubToken(value: string): boolean {
  const t = value.trim();
  if (!t) return false;
  return t.startsWith('ghp_') || t.startsWith('github_pat_') || t.startsWith('gho_');
}

/** Pi SSH password: Environment field, install override, or session state. */
function sshPassword(): string {
  return val('sshPasswordOverride') || val('sshPassword') || state.sshPassword;
}

/** GitHub PAT for agent artifact download (new input, or saved config / session). */
function githubToken(): string {
  const entered = val('githubToken');
  if (isNewGitHubToken(entered)) return entered.trim();
  return state.githubToken;
}

function val(id: string): string {
  return (document.getElementById(id) as HTMLInputElement)?.value?.trim() || '';
}

function setMsg(text: string, isError = false) {
  const el = document.getElementById('msg');
  if (!el) return;
  el.className = isError ? 'error' : text ? 'success' : '';
  el.textContent = text;
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}

init();
