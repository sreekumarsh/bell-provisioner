import './app.css';
import {
  GetConfig,
  SaveConfig,
  GetSession,
  Login,
  Logout,
  DiscoverDevices,
  ListDeviceTypes,
  ListCapabilities,
  ListDeviceFamilies,
  CreateCapability,
  CreateDeviceFamily,
  CreateDeviceType,
  ApplyRegistry,
  ListEntitlements,
  CreateEntitlement,
  PatchEntitlement,
  ListPlans,
  CreatePlan,
  PatchPlan,
  Provision,
  Install,
  Verify,
  TestSSH,
} from '../wailsjs/go/main/App';
import { config } from '../wailsjs/go/models';
import type { gateway, main } from '../wailsjs/go/models';

type View = 'login' | 'dashboard' | 'provision' | 'registry' | 'subscriptions';
type ProvisionStep = 1 | 2 | 3 | 4;
type RegistryTab = 'capabilities' | 'families' | 'types';
type SubscriptionsTab = 'entitlements' | 'plans';

const ENTITLEMENT_CATEGORIES: { value: string; label: string }[] = [
  { value: '', label: '— none —' },
  { value: 'recording', label: 'Recording' },
  { value: 'sharing', label: 'Sharing' },
  { value: 'ai', label: 'AI' },
  { value: 'support', label: 'Support' },
];

const ENTITLEMENT_UNITS: { value: string; label: string }[] = [
  { value: '', label: '— none —' },
  { value: 'days', label: 'Days' },
  { value: 'seconds', label: 'Seconds' },
  { value: 'count', label: 'Count' },
  { value: 'kbps', label: 'Kbps' },
];

const provisionLabels = ['Environment', 'Discover', 'Provision', 'Install'];

let view: View = 'login';
let provisionStep: ProvisionStep = 1;
let registryTab: RegistryTab = 'types';
let subscriptionsTab: SubscriptionsTab = 'entitlements';
let loginUser: main.LoginResult | null = null;
let selectedHost = '';
let provisionResult: main.ProvisionResult | null = null;
let savedCfg: config.AppConfig | null = null;

let deviceTypes: gateway.DeviceType[] = [];
let capabilities: gateway.Capability[] = [];
let families: gateway.DeviceFamily[] = [];
let entitlements: gateway.Entitlement[] = [];
let plans: gateway.Plan[] = [];
let editingEntitlementId: string | null = null;
let editingPlanId: string | null = null;
let planEntitlementDrafts: { key: string; value: string }[] = [{ key: '', value: 'true' }];
let editPlanEntitlementDrafts: { key: string; value: string }[] = [{ key: '', value: 'true' }];

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
  dtid: '',
  factoryDeviceID: '',
  serial: '',
  hwVersion: '1.1',
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
    state.sshUser = (cfg.ssh_user || state.sshUser || 'pi').toLowerCase();
    state.sshHost = cfg.ssh_host || 'raspberrypi.local';
    state.sshPort = cfg.ssh_port || 22;
    state.macIP = cfg.mac_ip || '';
    state.phone = cfg.phone || '';
    state.githubToken = cfg.github_token || '';
    state.sshPassword = cfg.ssh_password || '';
  } catch (e) {
    console.error(e);
  }
  try {
    const session = await GetSession();
    if (session.logged_in) {
      loginUser = {
        user_id: '',
        name: session.user_name,
        phone: session.phone,
        role: session.role,
        is_admin: session.role === 'admin',
      };
      view = 'dashboard';
    }
  } catch (e) {
    console.error(e);
  }
  render();
}

function render() {
  const app = document.getElementById('app')!;
  app.innerHTML = `
    <header class="app-header">
      <div>
        <h1>Bell Provisioner</h1>
        <p class="subtitle">Factory / lab device provisioning</p>
      </div>
      ${renderHeaderActions()}
    </header>
    ${view === 'provision' ? renderProvisionSteps() : ''}
    ${view === 'registry' ? renderRegistryTabs() : ''}
    ${view === 'subscriptions' ? renderSubscriptionsTabs() : ''}
    <div class="panel">${renderPanel()}</div>
  `;
  bindEvents();
}

function renderHeaderActions(): string {
  if (view === 'login') return '';
  const who = loginUser?.name || loginUser?.phone || state.phone;
  return `
    <div class="header-actions">
      <span class="user-badge">${esc(who)} · ${esc(state.gatewayURL)}</span>
      <button class="secondary" id="btnLogout">Logout</button>
    </div>`;
}

function renderProvisionSteps(): string {
  return `<div class="steps">${provisionLabels.map((label, i) => {
    const n = (i + 1) as ProvisionStep;
    let cls = 'step-indicator';
    if (n === provisionStep) cls += ' active';
    else if (n < provisionStep) cls += ' done';
    return `<div class="${cls}">${n}. ${label}</div>`;
  }).join('')}</div>`;
}

function renderRegistryTabs(): string {
  const tabs: { id: RegistryTab; label: string }[] = [
    { id: 'capabilities', label: 'Capabilities' },
    { id: 'families', label: 'Families' },
    { id: 'types', label: 'Device types' },
  ];
  return `<div class="tabs">${tabs.map(t =>
    `<button class="tab ${registryTab === t.id ? 'active' : ''}" data-tab="${t.id}">${t.label}</button>`
  ).join('')}</div>`;
}

function renderSubscriptionsTabs(): string {
  const tabs: { id: SubscriptionsTab; label: string }[] = [
    { id: 'entitlements', label: 'Entitlements' },
    { id: 'plans', label: 'Plans' },
  ];
  return `<div class="tabs">${tabs.map(t =>
    `<button class="tab ${subscriptionsTab === t.id ? 'active' : ''}" data-subtab="${t.id}">${t.label}</button>`
  ).join('')}</div>`;
}

function renderPanel(): string {
  switch (view) {
    case 'login': return renderLogin();
    case 'dashboard': return renderDashboard();
    case 'provision':
      switch (provisionStep) {
        case 1: return renderEnv();
        case 2: return renderDiscover();
        case 3: return renderProvision();
        case 4: return renderInstall();
      }
      break;
    case 'registry':
      switch (registryTab) {
        case 'capabilities': return renderRegistryCapabilities();
        case 'families': return renderRegistryFamilies();
        case 'types': return renderRegistryTypes();
      }
      break;
    case 'subscriptions':
      switch (subscriptionsTab) {
        case 'entitlements': return renderSubscriptionsEntitlements();
        case 'plans': return renderSubscriptionsPlans();
      }
  }
  return '';
}

function renderLogin(): string {
  return `
    <h2>Admin login</h2>
    <p class="hint">Sign in first. After login you can manage the device registry or start a provisioning run.</p>
    <div class="field">
      <label>Gateway URL</label>
      <input id="gatewayURL" value="${esc(state.gatewayURL)}" placeholder="https://api.vyooham.com" />
    </div>
    <div class="field">
      <label>Phone (+country code)</label>
      <input id="phone" value="${esc(state.phone)}" placeholder="+919876543210" />
    </div>
    <div class="field">
      <label>Password</label>
      <input id="password" type="password" />
    </div>
    <div class="actions">
      <button class="primary" id="btnLogin">Login</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderDashboard(): string {
  return `
    <h2>Dashboard</h2>
    <p class="hint">Welcome${loginUser?.name ? `, ${esc(loginUser.name)}` : ''}. Choose an action below.</p>
    <div class="dashboard-grid">
      <button class="dashboard-card" id="btnStartProvision">
        <strong>Start provisioning</strong>
        <span>Configure environment, discover a Pi, register in cloud, and install credentials.</span>
      </button>
      <button class="dashboard-card" id="btnOpenRegistry">
        <strong>Device registry</strong>
        <span>Manage capabilities, families, and device types via admin API.</span>
      </button>
      <button class="dashboard-card" id="btnOpenSubscriptions">
        <strong>Entitlements &amp; plans</strong>
        <span>Define paid-feature catalog and priced plan bundles (Phase 1).</span>
      </button>
    </div>
    <div id="msg"></div>
  `;
}

function renderEnv(): string {
  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
    </div>
    <h2>Environment</h2>
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
        <input id="sshUser" value="${esc(state.sshUser)}" placeholder="pi" />
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
      ${state.sshPassword ? '<p class="field-note success">SSH password saved in config</p>' : ''}
    </div>
    <div class="field">
      <label>GitHub token (read-only)</label>
      <input id="githubToken" type="password" autocomplete="off" placeholder="${state.githubToken ? 'leave blank to keep saved token' : 'ghp_… or github_pat_…'}" />
      ${state.githubToken ? '<p class="field-note success">GitHub token saved in config</p>' : ''}
    </div>
    <div class="actions">
      <button class="primary" id="btnNextEnv">Continue</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderDiscover(): string {
  const host = selectedHost || state.sshHost;
  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
    </div>
    <h2>Discover device</h2>
    <p class="hint">SSH host from Environment is pre-selected (<strong>${esc(host)}</strong>).</p>
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
  const typeOptions = deviceTypes.length
    ? deviceTypes
        .filter(t => !t.deprecated)
        .map(t => `<option value="${esc(t.dtid)}" ${state.dtid === t.dtid ? 'selected' : ''}>${esc(t.friendly_name || t.dtid)} (${esc(t.dtid)})</option>`)
        .join('')
    : '<option value="">No types — add one in Device registry</option>';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
    </div>
    <h2>Provision in cloud (v2)</h2>
    <div class="field">
      <label>Device type (DTID)</label>
      <select id="dtid">${typeOptions}</select>
    </div>
    <div class="field">
      <label>Factory device_id</label>
      <input id="factoryDeviceID" value="${esc(state.factoryDeviceID)}" placeholder="00042" />
    </div>
    <div class="field">
      <label>Serial number</label>
      <input id="serial" value="${esc(state.serial)}" placeholder="DB-2605-0042" />
    </div>
    <div class="field">
      <label>Hardware version</label>
      <input id="hwVersion" value="${esc(state.hwVersion)}" />
    </div>
    <label class="checkbox-field">
      <input type="checkbox" id="overwriteSerial" ${state.overwriteSerial ? 'checked' : ''} />
      Overwrite if serial already provisioned
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
      <strong>global_device_id:</strong> <code>${esc(provisionResult.global_device_id)}</code><br/>
      <strong>DSID:</strong> <code>${esc(provisionResult.dsid)}</code><br/>
      <strong>DTID:</strong> <code>${esc(provisionResult.dtid)}</code> · unit <code>${esc(provisionResult.device_id)}</code>
    </div>
  ` : '';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
    </div>
    <h2>Install on device</h2>
    ${summary}
    <div class="field" style="margin-top:16px">
      <label>SSH password override (optional)</label>
      <input id="sshPasswordOverride" type="password" autocomplete="off" />
    </div>
    <label class="checkbox-field">
      <input type="checkbox" id="checkoutAgent" ${state.checkoutAgent ? 'checked' : ''} />
      Download latest CI agent build and install on Pi
    </label>
    <label class="checkbox-field">
      <input type="checkbox" id="deployAgentEnv" ${state.deployAgentEnv ? 'checked' : ''} />
      Deploy agent.env for ${state.backendProfile === 'mac' ? 'Mac LAN' : 'VPS'}
    </label>
    <div class="actions">
      <button class="secondary" id="btnBack">Back</button>
      <button class="primary" id="btnInstall">Install on Pi</button>
      <button class="primary" id="btnVerify">Verify MQTT</button>
    </div>
    <div id="verifyResult"></div>
    <div id="msg"></div>
  `;
}

function renderRegistryCapabilities(): string {
  const rows = capabilities.length
    ? capabilities.map(c => `
      <tr>
        <td><code>${esc(c.capid)}</code></td>
        <td>${esc(c.friendly_name)}</td>
        <td>${esc(c.layer)}</td>
        <td>${c.deprecated ? 'yes' : '—'}</td>
      </tr>`).join('')
    : '<tr><td colspan="4" class="empty">No capabilities yet</td></tr>';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
      <button class="secondary" id="btnRefreshRegistry">Refresh</button>
    </div>
    <h2>Capabilities</h2>
    <table class="registry-table">
      <thead><tr><th>CAPID</th><th>Name</th><th>Layer</th><th>Deprecated</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    <h3 class="section-title">Add capability</h3>
    <div class="row">
      <div class="field"><label>CAPID</label><input id="newCapID" placeholder="cap_cam00001" /></div>
      <div class="field"><label>Name</label><input id="newCapName" placeholder="Camera" /></div>
    </div>
    <div class="row">
      <div class="field">
        <label>Layer</label>
        <select id="newCapLayer">
          <option value="intrinsic">intrinsic</option>
          <option value="runtime">runtime</option>
        </select>
      </div>
      <div class="field"><label>Description</label><input id="newCapDesc" placeholder="Optional" /></div>
    </div>
    <div class="actions">
      <button class="primary" id="btnCreateCap">Create capability</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderRegistryFamilies(): string {
  const rows = families.length
    ? families.map(f => `
      <tr>
        <td><code>${esc(f.dfid)}</code></td>
        <td>${esc(f.friendly_name)}</td>
        <td>${esc(f.description || '')}</td>
        <td>${f.deprecated ? 'yes' : '—'}</td>
      </tr>`).join('')
    : '<tr><td colspan="4" class="empty">No families yet</td></tr>';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
      <button class="secondary" id="btnRefreshRegistry">Refresh</button>
    </div>
    <h2>Device families</h2>
    <table class="registry-table">
      <thead><tr><th>DFID</th><th>Name</th><th>Description</th><th>Deprecated</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    <h3 class="section-title">Add family</h3>
    <div class="row">
      <div class="field"><label>DFID</label><input id="newDFID" placeholder="df_door0001" /></div>
      <div class="field"><label>Name</label><input id="newFamilyName" placeholder="Doorbells" /></div>
    </div>
    <div class="field"><label>Description</label><input id="newFamilyDesc" placeholder="Optional" /></div>
    <div class="actions">
      <button class="primary" id="btnCreateFamily">Create family</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderRegistryTypes(): string {
  const rows = deviceTypes.length
    ? deviceTypes.map(t => `
      <tr>
        <td><code>${esc(t.dtid)}</code></td>
        <td>${esc(t.friendly_name)}</td>
        <td><code>${esc(t.dfid)}</code></td>
        <td>${(t.capabilities || []).map(c => `<code>${esc(c)}</code>`).join(' ') || '—'}</td>
        <td>${t.deprecated ? 'yes' : '—'}</td>
      </tr>`).join('')
    : '<tr><td colspan="5" class="empty">No device types yet</td></tr>';

  const familyOptions = families.map(f =>
    `<option value="${esc(f.dfid)}">${esc(f.friendly_name || f.dfid)}</option>`
  ).join('');

  const capChecks = capabilities.filter(c => !c.deprecated).map(c => `
    <label class="checkbox-field cap-check">
      <input type="checkbox" class="newTypeCap" value="${esc(c.capid)}" />
      <span><code>${esc(c.capid)}</code> · ${esc(c.friendly_name)}</span>
    </label>
  `).join('') || '<p class="hint">Create capabilities first.</p>';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
      <button class="secondary" id="btnRefreshRegistry">Refresh</button>
    </div>
    <h2>Device types</h2>
    <table class="registry-table">
      <thead><tr><th>DTID</th><th>Name</th><th>Family</th><th>Capabilities</th><th>Deprecated</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    <h3 class="section-title">Add device type</h3>
    <div class="row">
      <div class="field"><label>DTID</label><input id="newDTID" placeholder="dt_wired0002" /></div>
      <div class="field">
        <label>Family (DFID)</label>
        <select id="newTypeDFID">${familyOptions || '<option value="">Create a family first</option>'}</select>
      </div>
    </div>
    <div class="row">
      <div class="field"><label>Name</label><input id="newTypeName" placeholder="Wired doorbell v2" /></div>
      <div class="field"><label>Description</label><input id="newTypeDesc" placeholder="Optional" /></div>
    </div>
    <div class="field">
      <label>Capabilities</label>
      <div class="cap-grid">${capChecks}</div>
    </div>
    <div class="actions">
      <button class="primary" id="btnCreateType">Create device type</button>
    </div>
    <h3 class="section-title">Apply server registry YAML</h3>
    <p class="hint">Re-applies <code>registry/registry.yaml</code> on the device service (VPS). Use dry-run first.</p>
    <div class="actions">
      <button class="secondary" id="btnApplyDryRun">Dry run</button>
      <button class="primary" id="btnApplyRegistry">Apply YAML</button>
    </div>
    <div id="applyResult"></div>
    <div id="msg"></div>
  `;
}

function renderSubscriptionsEntitlements(): string {
  const editing = editingEntitlementId
    ? entitlements.find(e => e.entitlement_id === editingEntitlementId)
    : undefined;

  const rows = entitlements.length
    ? entitlements.map(e => `
      <tr>
        <td><code>${esc(e.key)}</code></td>
        <td>${esc(e.friendly_name)}</td>
        <td>${esc(e.scope)}</td>
        <td>${esc(e.value_type)}</td>
        <td>${esc(categoryLabel(e.category || ''))}</td>
        <td>${e.deprecated ? 'yes' : '—'}</td>
        <td><button type="button" class="link-btn btnEditEntitlement" data-ent-id="${esc(e.entitlement_id || '')}">Edit</button></td>
      </tr>`).join('')
    : '<tr><td colspan="7" class="empty">No entitlements yet</td></tr>';

  const editSection = editing ? `
    <h3 class="section-title">Edit entitlement</h3>
    <p class="hint">Key, scope, value type, and unit cannot be changed after creation.</p>
    <div class="row">
      <div class="field"><label>Key</label><input value="${esc(editing.key)}" disabled /></div>
      <div class="field"><label>Name</label><input id="editEntName" value="${esc(editing.friendly_name)}" /></div>
    </div>
    <div class="row">
      <div class="field"><label>Scope</label><input value="${esc(editing.scope)}" disabled /></div>
      <div class="field"><label>Value type</label><input value="${esc(editing.value_type)}" disabled /></div>
      <div class="field">
        <label>Category</label>
        <select id="editEntCategory">${renderCategoryOptions(editing.category || '')}</select>
      </div>
    </div>
    <div class="field"><label>Description</label><input id="editEntDesc" value="${esc(editing.description || '')}" /></div>
    <div class="row">
      <div class="field"><label>Default value (JSON)</label><input id="editEntDefault" value="${esc(formatJSONValue(editing.default_value))}" /></div>
      <div class="field"><label>Unit</label><input value="${esc(unitLabel(editing.unit || ''))}" disabled /></div>
    </div>
    <div class="field">
      <label><input type="checkbox" id="editEntDeprecated" ${editing.deprecated ? 'checked' : ''} /> Deprecated</label>
    </div>
    <div class="actions">
      <button class="primary" id="btnSaveEntitlement">Save changes</button>
      <button class="secondary" id="btnCancelEditEntitlement">Cancel</button>
    </div>
  ` : '';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
      <button class="secondary" id="btnRefreshSubscriptions">Refresh</button>
    </div>
    <h2>Entitlements</h2>
    <p class="hint">Paid-feature catalog (auth-service). Distinct from hardware capabilities in device registry.</p>
    <table class="registry-table">
      <thead><tr><th>Key</th><th>Name</th><th>Scope</th><th>Type</th><th>Category</th><th>Deprecated</th><th></th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${editSection}
    <h3 class="section-title">Add entitlement</h3>
    <div class="row">
      <div class="field"><label>Key</label><input id="newEntKey" placeholder="cloud_recording" /></div>
      <div class="field"><label>Name</label><input id="newEntName" placeholder="Cloud recording" /></div>
    </div>
    <div class="row">
      <div class="field">
        <label>Scope</label>
        <select id="newEntScope">
          <option value="device">device</option>
          <option value="location">location</option>
          <option value="account">account</option>
          <option value="all">all</option>
        </select>
      </div>
      <div class="field">
        <label>Value type</label>
        <select id="newEntValueType">
          <option value="boolean">boolean</option>
          <option value="number">number</option>
          <option value="enum">enum</option>
          <option value="string">string</option>
        </select>
      </div>
      <div class="field">
        <label>Category</label>
        <select id="newEntCategory">${renderCategoryOptions('recording')}</select>
      </div>
    </div>
    <div class="field"><label>Description</label><input id="newEntDesc" placeholder="Optional" /></div>
    <div class="row">
      <div class="field"><label>Default value (JSON)</label><input id="newEntDefault" value="false" placeholder="false" /></div>
      <div class="field">
        <label>Unit</label>
        <select id="newEntUnit">${renderUnitOptions('')}</select>
      </div>
    </div>
    <div class="actions">
      <button class="primary" id="btnCreateEntitlement">Create entitlement</button>
    </div>
    <div id="msg"></div>
  `;
}

function renderSubscriptionsPlans(): string {
  const editing = editingPlanId
    ? plans.find(p => p.plan_id === editingPlanId)
    : undefined;

  const rows = plans.length
    ? plans.map(p => `
      <tr>
        <td><code>${esc(p.code)}</code></td>
        <td>${esc(p.friendly_name)}</td>
        <td>${esc(p.subject_type)}</td>
        <td>${formatPrice(p.price_amount_minor, p.price_currency)}</td>
        <td>${esc(p.billing_interval)}</td>
        <td>${esc(p.status)}</td>
        <td>${formatPlanEntitlements(p.entitlements)}</td>
        <td><button type="button" class="link-btn btnEditPlan" data-plan-id="${esc(p.plan_id || '')}">Edit</button></td>
      </tr>`).join('')
    : '<tr><td colspan="8" class="empty">No plans yet</td></tr>';

  const editSection = editing ? `
    <h3 class="section-title">Edit plan</h3>
    <p class="hint">Code and subject type cannot be changed after creation.</p>
    <div class="row">
      <div class="field"><label>Code</label><input value="${esc(editing.code)}" disabled /></div>
      <div class="field"><label>Name</label><input id="editPlanName" value="${esc(editing.friendly_name)}" /></div>
    </div>
    <div class="row">
      <div class="field"><label>Subject type</label><input value="${esc(editing.subject_type)}" disabled /></div>
      <div class="field">
        <label>Billing interval</label>
        <select id="editPlanInterval">
          ${renderBillingIntervalOptions(editing.billing_interval)}
        </select>
      </div>
      <div class="field"><label>Price (minor units)</label><input id="editPlanPrice" type="number" value="${editing.price_amount_minor}" /></div>
      <div class="field"><label>Currency</label><input id="editPlanCurrency" value="${esc(editing.price_currency)}" /></div>
    </div>
    <div class="row">
      <div class="field">
        <label>Status</label>
        <select id="editPlanStatus">
          <option value="active" ${editing.status === 'active' ? 'selected' : ''}>active</option>
          <option value="inactive" ${editing.status === 'inactive' ? 'selected' : ''}>inactive</option>
        </select>
      </div>
      <div class="field"><label>Trial days</label><input id="editPlanTrialDays" type="number" value="${editing.trial_days ?? 0}" /></div>
    </div>
    <div class="field"><label>Description</label><input id="editPlanDesc" value="${esc(editing.description || '')}" /></div>
    <h4 class="section-title">Entitlements in this plan</h4>
    <div id="editPlanEntitlementsList">${renderPlanEntitlementRows(editPlanEntitlementDrafts, 'edit')}</div>
    <div class="actions">
      <button type="button" class="secondary" id="btnAddEditPlanEnt">Add entitlement</button>
    </div>
    <div class="actions">
      <button class="primary" id="btnSavePlan">Save changes</button>
      <button class="secondary" id="btnCancelEditPlan">Cancel</button>
    </div>
  ` : '';

  return `
    <div class="panel-toolbar">
      <button class="link-btn" id="btnBackDashboard">← Dashboard</button>
      <button class="secondary" id="btnRefreshSubscriptions">Refresh</button>
    </div>
    <h2>Plans</h2>
    <p class="hint">Priced bundles of entitlements. Phase 1 defines catalog only — no runtime assignment yet.</p>
    <table class="registry-table">
      <thead><tr><th>Code</th><th>Name</th><th>Subject</th><th>Price</th><th>Interval</th><th>Status</th><th>Entitlements</th><th></th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${editSection}
    <h3 class="section-title">Add plan</h3>
    <div class="row">
      <div class="field"><label>Code</label><input id="newPlanCode" placeholder="cloud_standard" /></div>
      <div class="field"><label>Name</label><input id="newPlanName" placeholder="Cloud Standard" /></div>
    </div>
    <div class="row">
      <div class="field">
        <label>Subject type</label>
        <select id="newPlanSubject">
          <option value="device">device</option>
          <option value="location">location</option>
          <option value="account">account</option>
        </select>
      </div>
      <div class="field">
        <label>Billing interval</label>
        <select id="newPlanInterval">
          <option value="month">month</option>
          <option value="year">year</option>
          <option value="one_time">one_time</option>
          <option value="lifetime">lifetime</option>
        </select>
      </div>
      <div class="field"><label>Price (minor units)</label><input id="newPlanPrice" type="number" value="14900" /></div>
      <div class="field"><label>Currency</label><input id="newPlanCurrency" value="INR" /></div>
    </div>
    <div class="field"><label>Description</label><input id="newPlanDesc" placeholder="Optional" /></div>
    <h4 class="section-title">Entitlements in this plan</h4>
    <p class="hint">Attach one or more catalog entitlements with a JSON value each (e.g. <code>true</code>, <code>30</code>).</p>
    <div id="planEntitlementsList">${renderPlanEntitlementRows(planEntitlementDrafts, 'create')}</div>
    <div class="actions">
      <button type="button" class="secondary" id="btnAddPlanEnt">Add entitlement</button>
    </div>
    <div class="actions">
      <button class="primary" id="btnCreatePlan">Create plan</button>
    </div>
    <div id="msg"></div>
  `;
}

function formatPrice(minor: number, currency: string): string {
  if (!minor) return 'Free';
  const major = (minor / 100).toFixed(2);
  return `${currency || 'INR'} ${major}`;
}

function renderEntitlementKeyOptions(selected: string): string {
  return entitlements.filter(e => !e.deprecated).map(e =>
    `<option value="${esc(e.key)}" ${e.key === selected ? 'selected' : ''}>${esc(e.friendly_name || e.key)} (${esc(e.value_type)})</option>`
  ).join('');
}

function renderBillingIntervalOptions(selected: string): string {
  const intervals = ['month', 'year', 'one_time', 'lifetime'];
  return intervals.map(i =>
    `<option value="${i}" ${i === selected ? 'selected' : ''}>${i}</option>`
  ).join('');
}

function renderPlanEntitlementRows(drafts: { key: string; value: string }[], list: 'create' | 'edit'): string {
  if (drafts.length === 0) {
    drafts.push({ key: '', value: 'true' });
  }
  return drafts.map((row, index) => `
    <div class="row plan-ent-row" data-list="${list}">
      <div class="field">
        <label>Entitlement</label>
        <select class="planEntKey" data-list="${list}" data-index="${index}">
          <option value="">— select —</option>
          ${renderEntitlementKeyOptions(row.key)}
        </select>
      </div>
      <div class="field">
        <label>Value (JSON)</label>
        <input class="planEntValue" data-list="${list}" data-index="${index}" value="${esc(row.value)}" placeholder="true, 30, ..." />
      </div>
      ${drafts.length > 1
        ? `<button type="button" class="link-btn btnRemovePlanEnt" data-list="${list}" data-index="${index}">Remove</button>`
        : ''}
    </div>`).join('');
}

function formatPlanEntitlements(refs: gateway.PlanEntitlementRef[] | undefined): string {
  if (!refs?.length) return '—';
  return refs.map(e => `<code>${esc(e.key || '')}</code>=<code>${esc(e.value || '')}</code>`).join(' ');
}

function syncPlanEntitlementDraftsFromDOM(drafts: { key: string; value: string }[], list: 'create' | 'edit') {
  document.querySelectorAll(`.plan-ent-row[data-list="${list}"]`).forEach((row, index) => {
    if (!drafts[index]) return;
    drafts[index].key = (row.querySelector('.planEntKey') as HTMLSelectElement)?.value ?? '';
    drafts[index].value = (row.querySelector('.planEntValue') as HTMLInputElement)?.value ?? '';
  });
}

function collectPlanEntitlements(drafts: { key: string; value: string }[], list: 'create' | 'edit'): gateway.PlanEntitlementRef[] {
  syncPlanEntitlementDraftsFromDOM(drafts, list);
  const payload: gateway.PlanEntitlementRef[] = [];
  for (const row of drafts) {
    if (!row.key) continue;
    parseJSONDefault(row.value || 'null');
    payload.push({ key: row.key, value: row.value });
  }
  return payload;
}

function planEntitlementsFromPlan(plan: gateway.Plan): { key: string; value: string }[] {
  if (!plan.entitlements?.length) {
    return [{ key: '', value: 'true' }];
  }
  return plan.entitlements.map(e => ({
    key: e.key || '',
    value: e.value || 'true',
  }));
}

function parseTrialDays(raw: string): number {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    throw new Error('Trial days must be a non-negative whole number');
  }
  return parseInt(trimmed, 10);
}

function bindEvents() {
  document.getElementById('btnLogout')?.addEventListener('click', () => {
    void doLogout();
  });

  document.getElementById('btnLogin')?.addEventListener('click', () => void doLogin());

  document.getElementById('btnStartProvision')?.addEventListener('click', () => {
    provisionStep = 1;
    provisionResult = null;
    view = 'provision';
    render();
  });

  document.getElementById('btnOpenRegistry')?.addEventListener('click', () => {
    registryTab = 'types';
    view = 'registry';
    void loadRegistryData();
  });

  document.getElementById('btnOpenSubscriptions')?.addEventListener('click', () => {
    subscriptionsTab = 'entitlements';
    editingEntitlementId = null;
    editingPlanId = null;
    view = 'subscriptions';
    void loadSubscriptionsData();
  });

  document.getElementById('btnBackDashboard')?.addEventListener('click', () => {
    view = 'dashboard';
    render();
  });

  document.querySelectorAll('.tab').forEach(el => {
    el.addEventListener('click', () => {
      const subtab = el.getAttribute('data-subtab') as SubscriptionsTab | null;
      if (subtab) {
        subscriptionsTab = subtab;
        render();
        return;
      }
      registryTab = el.getAttribute('data-tab') as RegistryTab;
      render();
    });
  });

  document.getElementById('btnRefreshRegistry')?.addEventListener('click', () => void loadRegistryData());
  document.getElementById('btnRefreshSubscriptions')?.addEventListener('click', () => void loadSubscriptionsData());

  document.getElementById('btnCreateCap')?.addEventListener('click', () => void createCapability());
  document.getElementById('btnCreateFamily')?.addEventListener('click', () => void createFamily());
  document.getElementById('btnCreateType')?.addEventListener('click', () => void createDeviceType());
  document.getElementById('btnCreateEntitlement')?.addEventListener('click', () => void createEntitlement());
  document.getElementById('btnSaveEntitlement')?.addEventListener('click', () => void saveEntitlement());
  document.getElementById('btnCancelEditEntitlement')?.addEventListener('click', () => {
    editingEntitlementId = null;
    render();
  });
  document.querySelectorAll('.btnEditEntitlement').forEach(el => {
    el.addEventListener('click', () => {
      const id = el.getAttribute('data-ent-id');
      if (id) {
        editingEntitlementId = id;
        render();
      }
    });
  });
  document.getElementById('btnCreatePlan')?.addEventListener('click', () => void createPlan());
  document.getElementById('btnSavePlan')?.addEventListener('click', () => void savePlan());
  document.getElementById('btnCancelEditPlan')?.addEventListener('click', () => {
    editingPlanId = null;
    render();
  });
  document.querySelectorAll('.btnEditPlan').forEach(el => {
    el.addEventListener('click', () => {
      const id = el.getAttribute('data-plan-id');
      if (!id) return;
      const plan = plans.find(p => p.plan_id === id);
      if (!plan) return;
      editingPlanId = id;
      editPlanEntitlementDrafts = planEntitlementsFromPlan(plan);
      render();
    });
  });
  document.getElementById('btnAddPlanEnt')?.addEventListener('click', () => {
    syncPlanEntitlementDraftsFromDOM(planEntitlementDrafts, 'create');
    planEntitlementDrafts.push({ key: '', value: 'true' });
    render();
  });
  document.getElementById('btnAddEditPlanEnt')?.addEventListener('click', () => {
    syncPlanEntitlementDraftsFromDOM(editPlanEntitlementDrafts, 'edit');
    editPlanEntitlementDrafts.push({ key: '', value: 'true' });
    render();
  });
  document.querySelectorAll('.btnRemovePlanEnt').forEach(btn => {
    btn.addEventListener('click', () => {
      const list = btn.getAttribute('data-list') as 'create' | 'edit' | null;
      const index = Number(btn.getAttribute('data-index'));
      const drafts = list === 'edit' ? editPlanEntitlementDrafts : planEntitlementDrafts;
      if (!list || Number.isNaN(index)) return;
      syncPlanEntitlementDraftsFromDOM(drafts, list);
      drafts.splice(index, 1);
      if (drafts.length === 0) {
        drafts.push({ key: '', value: 'true' });
      }
      render();
    });
  });
  document.querySelectorAll('.planEntKey').forEach(el => {
    el.addEventListener('change', (e) => {
      const list = (e.target as HTMLSelectElement).dataset.list as 'create' | 'edit' | undefined;
      const index = Number((e.target as HTMLSelectElement).dataset.index);
      const key = (e.target as HTMLSelectElement).value;
      const drafts = list === 'edit' ? editPlanEntitlementDrafts : planEntitlementDrafts;
      if (!list || Number.isNaN(index) || !drafts[index]) return;
      drafts[index].key = key;
      const ent = entitlements.find(item => item.key === key);
      if (ent?.default_value) {
        drafts[index].value = ent.default_value;
      }
    });
  });
  document.getElementById('btnApplyDryRun')?.addEventListener('click', () => void runApplyRegistry(true));
  document.getElementById('btnApplyRegistry')?.addEventListener('click', () => void runApplyRegistry(false));

  document.getElementById('btnBack')?.addEventListener('click', () => {
    if (view === 'provision' && provisionStep > 1) {
      provisionStep = (provisionStep - 1) as ProvisionStep;
      render();
    }
  });

  document.getElementById('btnNextEnv')?.addEventListener('click', () => void saveEnvAndContinue());

  document.getElementById('backendProfile')?.addEventListener('change', (e) => {
    state.backendProfile = (e.target as HTMLSelectElement).value;
    document.getElementById('macIPField')?.classList.toggle('hidden', state.backendProfile !== 'mac');
  });

  document.getElementById('btnProbeHost')?.addEventListener('click', () => void runDiscover(false));
  document.getElementById('btnScan')?.addEventListener('click', () => void runDiscover(true));

  document.getElementById('btnTestSSH')?.addEventListener('click', () => void testSSH());

  document.getElementById('btnNextDiscover')?.addEventListener('click', () => {
    selectedHost = val('selectedHost') || selectedHost || state.sshHost;
    if (!selectedHost) {
      setMsg('Enter a device host', true);
      return;
    }
    state.sshHost = selectedHost;
    provisionStep = 3;
    void loadDeviceTypes();
    render();
  });

  document.getElementById('btnProvision')?.addEventListener('click', () => void doProvision());

  document.getElementById('btnInstall')?.addEventListener('click', () => void doInstall());
  document.getElementById('btnVerify')?.addEventListener('click', () => void doVerify());
}

async function doLogin() {
  state.gatewayURL = val('gatewayURL') || state.gatewayURL;
  state.phone = val('phone');
  state.password = val('password');
  setMsg('Logging in…');
  try {
    await SaveConfig(buildAppConfig({ gateway_url: state.gatewayURL, phone: state.phone }));
    loginUser = await Login(state.phone, state.password);
    view = 'dashboard';
    setMsg('');
    render();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function doLogout() {
  Logout();
  loginUser = null;
  view = 'login';
  render();
}

async function saveEnvAndContinue() {
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
  provisionStep = 2;
  initDiscoverStep();
  render();
}

async function loadSubscriptionsData() {
  setMsg('Loading entitlements & plans…');
  let entErr = '';
  let planErr = '';
  try {
    entitlements = (await ListEntitlements()) ?? [];
  } catch (e) {
    entitlements = [];
    entErr = errorMessage(e);
  }
  try {
    plans = (await ListPlans()) ?? [];
  } catch (e) {
    plans = [];
    planErr = errorMessage(e);
  }
  if (view === 'subscriptions') render();
  if (entErr && planErr) {
    setMsg(formatRegistryError(`Entitlements: ${entErr}; Plans: ${planErr}`), true);
  } else if (entErr) {
    setMsg(formatRegistryError(`Entitlements: ${entErr}`), true);
  } else if (planErr) {
    setMsg(formatRegistryError(`Plans: ${planErr}`), true);
  } else {
    setMsg('');
  }
}

function formatJSONValue(v: unknown): string {
  if (v === undefined || v === null) return '';
  if (typeof v === 'string') return v;
  return JSON.stringify(v);
}

function errorMessage(e: unknown): string {
  if (e == null) return 'Unknown error';
  if (typeof e === 'string') return e;
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && 'message' in e && typeof (e as { message: unknown }).message === 'string') {
    return (e as { message: string }).message;
  }
  return String(e);
}

function categoryLabel(value: string): string {
  if (!value) return '—';
  const hit = ENTITLEMENT_CATEGORIES.find(c => c.value === value);
  return hit?.label ?? value;
}

function renderCategoryOptions(selected: string): string {
  const known = new Set(ENTITLEMENT_CATEGORIES.map(c => c.value));
  let html = ENTITLEMENT_CATEGORIES.map(c =>
    `<option value="${esc(c.value)}" ${c.value === selected ? 'selected' : ''}>${esc(c.label)}</option>`
  ).join('');
  if (selected && !known.has(selected)) {
    html += `<option value="${esc(selected)}" selected>${esc(selected)} (legacy)</option>`;
  }
  return html;
}

function unitLabel(value: string): string {
  if (!value) return '—';
  const hit = ENTITLEMENT_UNITS.find(u => u.value === value);
  return hit?.label ?? value;
}

function renderUnitOptions(selected: string): string {
  const known = new Set(ENTITLEMENT_UNITS.map(u => u.value));
  let html = ENTITLEMENT_UNITS.map(u =>
    `<option value="${esc(u.value)}" ${u.value === selected ? 'selected' : ''}>${esc(u.label)}</option>`
  ).join('');
  if (selected && !known.has(selected)) {
    html += `<option value="${esc(selected)}" selected>${esc(selected)} (legacy)</option>`;
  }
  return html;
}

function parseJSONDefault(raw: string): unknown {
  try {
    return JSON.parse(raw);
  } catch {
    throw new Error('Default value must be valid JSON (e.g. false, 0, "text")');
  }
}

function parsePriceMinor(raw: string): number {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    throw new Error('Price must be a non-negative whole number (minor units)');
  }
  return parseInt(trimmed, 10);
}

async function createEntitlement() {
  setMsg('Creating entitlement…');
  const defaultRaw = val('newEntDefault') || 'false';
  try {
    parseJSONDefault(defaultRaw);
    await CreateEntitlement({
      key: val('newEntKey'),
      friendly_name: val('newEntName'),
      description: val('newEntDesc'),
      scope: val('newEntScope') || 'device',
      value_type: val('newEntValueType') || 'boolean',
      unit: val('newEntUnit'),
      default_value: defaultRaw,
      category: val('newEntCategory'),
    } as gateway.Entitlement);
    setMsg('Entitlement created', false);
    await loadSubscriptionsData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function saveEntitlement() {
  if (!editingEntitlementId) return;
  setMsg('Saving entitlement…');
  const defaultRaw = val('editEntDefault');
  try {
    parseJSONDefault(defaultRaw);
    await PatchEntitlement(editingEntitlementId, {
      friendly_name: val('editEntName'),
      description: val('editEntDesc'),
      default_value: defaultRaw,
      category: val('editEntCategory'),
      deprecated: (document.getElementById('editEntDeprecated') as HTMLInputElement)?.checked ?? false,
    } as gateway.EntitlementPatch);
    editingEntitlementId = null;
    setMsg('Entitlement updated', false);
    await loadSubscriptionsData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function createPlan() {
  setMsg('Creating plan…');
  let entitlementsPayload: gateway.PlanEntitlementRef[];
  try {
    entitlementsPayload = collectPlanEntitlements(planEntitlementDrafts, 'create');
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
    return;
  }
  try {
    await CreatePlan({
      code: val('newPlanCode'),
      friendly_name: val('newPlanName'),
      description: val('newPlanDesc'),
      subject_type: val('newPlanSubject') || 'device',
      price_amount_minor: parsePriceMinor(val('newPlanPrice') || '0'),
      price_currency: val('newPlanCurrency') || 'INR',
      billing_interval: val('newPlanInterval') || 'month',
      trial_days: 0,
      status: 'active',
      entitlements: entitlementsPayload,
    } as gateway.Plan);
    planEntitlementDrafts = [{ key: '', value: 'true' }];
    setMsg('Plan created', false);
    await loadSubscriptionsData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function savePlan() {
  if (!editingPlanId) return;
  setMsg('Saving plan…');
  let entitlementsPayload: gateway.PlanEntitlementRef[];
  try {
    entitlementsPayload = collectPlanEntitlements(editPlanEntitlementDrafts, 'edit');
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
    return;
  }
  try {
    await PatchPlan(editingPlanId, {
      friendly_name: val('editPlanName'),
      description: val('editPlanDesc'),
      price_amount_minor: parsePriceMinor(val('editPlanPrice') || '0'),
      price_currency: val('editPlanCurrency') || 'INR',
      billing_interval: val('editPlanInterval') || 'month',
      trial_days: parseTrialDays(val('editPlanTrialDays') || '0'),
      status: val('editPlanStatus') || 'active',
      entitlements: entitlementsPayload,
    } as gateway.PlanPatch);
    editingPlanId = null;
    setMsg('Plan updated', false);
    await loadSubscriptionsData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function loadRegistryData() {
  setMsg('Loading registry…');
  try {
    const [caps, fams, types] = await Promise.all([
      ListCapabilities(),
      ListDeviceFamilies(),
      ListDeviceTypes(),
    ]);
    capabilities = caps ?? [];
    families = fams ?? [];
    deviceTypes = types ?? [];
    if (view === 'registry') render();
    setMsg('');
  } catch (e: any) {
    if (view === 'registry') render();
    setMsg(formatRegistryError(e?.message || 'Failed to load registry'), true);
  }
}

async function createCapability() {
  setMsg('Creating capability…');
  try {
    await CreateCapability({
      capid: val('newCapID'),
      friendly_name: val('newCapName'),
      layer: val('newCapLayer') || 'intrinsic',
      description: val('newCapDesc'),
      deprecated: false,
    });
    setMsg('Capability created', false);
    await loadRegistryData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function createFamily() {
  setMsg('Creating family…');
  try {
    await CreateDeviceFamily({
      dfid: val('newDFID'),
      friendly_name: val('newFamilyName'),
      description: val('newFamilyDesc'),
      deprecated: false,
    });
    setMsg('Family created', false);
    await loadRegistryData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function createDeviceType() {
  const caps = Array.from(document.querySelectorAll<HTMLInputElement>('.newTypeCap:checked'))
    .map(el => el.value);
  setMsg('Creating device type…');
  try {
    await CreateDeviceType({
      dtid: val('newDTID'),
      dfid: val('newTypeDFID'),
      friendly_name: val('newTypeName'),
      description: val('newTypeDesc'),
      capabilities: caps,
      deprecated: false,
    });
    setMsg('Device type created', false);
    await loadRegistryData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function runApplyRegistry(dryRun: boolean) {
  setMsg(dryRun ? 'Running dry-run…' : 'Applying registry…');
  try {
    const result = await ApplyRegistry(dryRun);
    const el = document.getElementById('applyResult');
    if (el) {
      el.innerHTML = `
        <div class="summary" style="margin-top:12px">
          <strong>Added (${result.added?.length || 0}):</strong> ${(result.added || []).join(', ') || '—'}<br/>
          <strong>Updated (${result.updated?.length || 0}):</strong> ${(result.updated || []).join(', ') || '—'}<br/>
          <strong>Rejected (${result.rejected?.length || 0}):</strong> ${(result.rejected || []).join('; ') || '—'}
        </div>`;
    }
    setMsg(dryRun ? 'Dry-run complete' : 'Registry apply complete', false);
    await loadRegistryData();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function loadDeviceTypes() {
  try {
    deviceTypes = (await ListDeviceTypes()) ?? [];
    if (!state.dtid && deviceTypes.length) {
      const active = deviceTypes.find(t => !t.deprecated);
      if (active) state.dtid = active.dtid;
    }
    if (view === 'provision' && provisionStep === 3) render();
  } catch (e: any) {
    deviceTypes = [];
    if (view === 'provision' && provisionStep === 3) {
      render();
      setMsg(formatRegistryError(e?.message || 'Failed to load device types'), true);
    }
  }
}

function formatRegistryError(message: string): string {
  const lower = message.toLowerCase();
  if (lower.includes('login required') || lower.includes('missing_token') || lower.includes('invalid or has expired') || lower.includes('token_expired')) {
    return 'Session expired — use Logout, log in again, then open the provision step';
  }
  if (lower.includes('forbidden') || lower.includes('admin access')) {
    return 'Admin access required — your account needs role=admin on the gateway';
  }
  return message;
}

function initDiscoverStep() {
  selectedHost = state.sshHost;
  if (!state.manualHost) state.manualHost = state.sshHost;
}

async function doProvision() {
  state.dtid = val('dtid');
  state.factoryDeviceID = val('factoryDeviceID');
  state.serial = val('serial');
  state.hwVersion = val('hwVersion') || '1.1';
  state.overwriteSerial = (document.getElementById('overwriteSerial') as HTMLInputElement)?.checked ?? true;
  if (!state.dtid) {
    setMsg('Select a device type — add one in Device registry if empty', true);
    return;
  }
  if (!state.factoryDeviceID) {
    setMsg('Factory device_id is required', true);
    return;
  }
  setMsg('Provisioning…');
  try {
    provisionResult = await Provision({
      dtid: state.dtid,
      device_id: state.factoryDeviceID,
      serial: state.serial,
      hw_version: state.hwVersion,
      overwrite: state.overwriteSerial,
    });
    provisionStep = 4;
    render();
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function doInstall() {
  const pw = sshPassword();
  if (!pw) {
    setMsg('SSH password is required', true);
    return;
  }
  const gh = githubToken();
  if (state.checkoutAgent && !gh) {
    setMsg('GitHub token is required (Environment step)', true);
    return;
  }
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
      github_token: gh,
    });
    const depsOk = result.ffmpeg_ok && result.go2rtc_ok && result.motion_ok;
    const suffix = [
      depsOk ? 'deps OK' : '',
      result.setup_server_ok ? 'Setup mode (:4444)' : '',
      result.agent_active ? 'Agent active' : '',
    ].filter(Boolean).join(' · ');
    setMsg(result.message + (suffix ? ' · ' + suffix : ''), !depsOk || !result.setup_server_ok || !result.agent_active);
  } catch (e: any) {
    setMsg(e?.message || String(e), true);
  }
}

async function doVerify() {
  setMsg('Verifying MQTT…');
  try {
    const result = await Verify({
      device_id: provisionResult?.global_device_id || '',
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
}

async function testSSH() {
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
}

async function runDiscover(fullLAN: boolean) {
  state.manualHost = val('manualHost');
  const list = document.getElementById('deviceList');
  if (!list) return;
  list.innerHTML = fullLAN
    ? '<li class="spinner">Scanning LAN…</li>'
    : '<li class="spinner">Probing configured host…</li>';
  setMsg('');
  try {
    const devices = (await DiscoverDevices(state.manualHost, fullLAN)) ?? [];
    if (devices.length === 0) {
      list.innerHTML = '<li style="cursor:default;color:var(--muted)">No SSH on probed hosts</li>';
      return;
    }
    list.innerHTML = devices.map(d => `
      <li data-host="${esc(d.host)}" data-serial="${esc(d.serial_number || '')}">
        <strong>${esc(d.host)}</strong>
        <div class="meta">SSH: ${d.ssh_reachable ? '✓' : '✗'} · Setup: ${d.setup_server ? '✓' : '✗'}</div>
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

function sshPassword(): string {
  return val('sshPasswordOverride') || val('sshPassword') || state.sshPassword;
}

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
