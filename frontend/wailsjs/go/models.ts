export namespace config {
	
	export class AppConfig {
	    gateway_url: string;
	    backend_profile: string;
	    ssh_user: string;
	    ssh_host: string;
	    ssh_port: number;
	    mac_ip: string;
	    phone: string;
	    github_token?: string;
	    ssh_password?: string;
	    agent_repo_url: string;
	    agent_repo_branch: string;
	    agent_repo_path: string;
	    agent_artifact_name: string;
	    access_token?: string;
	    refresh_token?: string;
	    token_expires_at?: number;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gateway_url = source["gateway_url"];
	        this.backend_profile = source["backend_profile"];
	        this.ssh_user = source["ssh_user"];
	        this.ssh_host = source["ssh_host"];
	        this.ssh_port = source["ssh_port"];
	        this.mac_ip = source["mac_ip"];
	        this.phone = source["phone"];
	        this.github_token = source["github_token"];
	        this.ssh_password = source["ssh_password"];
	        this.agent_repo_url = source["agent_repo_url"];
	        this.agent_repo_branch = source["agent_repo_branch"];
	        this.agent_repo_path = source["agent_repo_path"];
	        this.agent_artifact_name = source["agent_artifact_name"];
	        this.access_token = source["access_token"];
	        this.refresh_token = source["refresh_token"];
	        this.token_expires_at = source["token_expires_at"];
	    }
	}

}

export namespace device {
	
	export class InstallResult {
	    agent_active: boolean;
	    agent_checked_out: boolean;
	    ffmpeg_ok: boolean;
	    go2rtc_ok: boolean;
	    motion_ok: boolean;
	    setup_server_ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent_active = source["agent_active"];
	        this.agent_checked_out = source["agent_checked_out"];
	        this.ffmpeg_ok = source["ffmpeg_ok"];
	        this.go2rtc_ok = source["go2rtc_ok"];
	        this.motion_ok = source["motion_ok"];
	        this.setup_server_ok = source["setup_server_ok"];
	        this.message = source["message"];
	    }
	}

}

export namespace discover {
	
	export class Candidate {
	    host: string;
	    ssh_reachable: boolean;
	    setup_server: boolean;
	    serial_number?: string;
	    device_id?: string;
	    hw_version?: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.ssh_reachable = source["ssh_reachable"];
	        this.setup_server = source["setup_server"];
	        this.serial_number = source["serial_number"];
	        this.device_id = source["device_id"];
	        this.hw_version = source["hw_version"];
	    }
	}

}

export namespace gateway {
	
	export class Capability {
	    capid: string;
	    friendly_name: string;
	    layer: string;
	    description?: string;
	    deprecated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Capability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.capid = source["capid"];
	        this.friendly_name = source["friendly_name"];
	        this.layer = source["layer"];
	        this.description = source["description"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class DeviceFamily {
	    dfid: string;
	    friendly_name: string;
	    description?: string;
	    device_types?: string[];
	    deprecated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFamily(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dfid = source["dfid"];
	        this.friendly_name = source["friendly_name"];
	        this.description = source["description"];
	        this.device_types = source["device_types"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class DeviceType {
	    dtid: string;
	    dfid: string;
	    friendly_name: string;
	    capabilities?: string[];
	    description?: string;
	    deprecated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeviceType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dtid = source["dtid"];
	        this.dfid = source["dfid"];
	        this.friendly_name = source["friendly_name"];
	        this.capabilities = source["capabilities"];
	        this.description = source["description"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class Entitlement {
	    entitlement_id?: string;
	    key: string;
	    friendly_name: string;
	    description?: string;
	    scope: string;
	    value_type: string;
	    unit?: string;
	    default_value: string;
	    category?: string;
	    deprecated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entitlement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entitlement_id = source["entitlement_id"];
	        this.key = source["key"];
	        this.friendly_name = source["friendly_name"];
	        this.description = source["description"];
	        this.scope = source["scope"];
	        this.value_type = source["value_type"];
	        this.unit = source["unit"];
	        this.default_value = source["default_value"];
	        this.category = source["category"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class EntitlementPatch {
	    friendly_name: string;
	    description?: string;
	    default_value: string;
	    category?: string;
	    deprecated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EntitlementPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.friendly_name = source["friendly_name"];
	        this.description = source["description"];
	        this.default_value = source["default_value"];
	        this.category = source["category"];
	        this.deprecated = source["deprecated"];
	    }
	}
	export class PlanEntitlementRef {
	    key?: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new PlanEntitlementRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class Plan {
	    plan_id?: string;
	    code: string;
	    friendly_name: string;
	    description?: string;
	    subject_type: string;
	    price_amount_minor: number;
	    price_currency: string;
	    billing_interval: string;
	    trial_days: number;
	    status: string;
	    entitlements?: PlanEntitlementRef[];
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan_id = source["plan_id"];
	        this.code = source["code"];
	        this.friendly_name = source["friendly_name"];
	        this.description = source["description"];
	        this.subject_type = source["subject_type"];
	        this.price_amount_minor = source["price_amount_minor"];
	        this.price_currency = source["price_currency"];
	        this.billing_interval = source["billing_interval"];
	        this.trial_days = source["trial_days"];
	        this.status = source["status"];
	        this.entitlements = this.convertValues(source["entitlements"], PlanEntitlementRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PlanPatch {
	    friendly_name: string;
	    description?: string;
	    price_amount_minor: number;
	    price_currency: string;
	    billing_interval: string;
	    trial_days: number;
	    status: string;
	    entitlements: PlanEntitlementRef[];
	
	    static createFrom(source: any = {}) {
	        return new PlanPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.friendly_name = source["friendly_name"];
	        this.description = source["description"];
	        this.price_amount_minor = source["price_amount_minor"];
	        this.price_currency = source["price_currency"];
	        this.billing_interval = source["billing_interval"];
	        this.trial_days = source["trial_days"];
	        this.status = source["status"];
	        this.entitlements = this.convertValues(source["entitlements"], PlanEntitlementRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class ApplyRegistryResult {
	    added: string[];
	    updated: string[];
	    rejected: string[];
	
	    static createFrom(source: any = {}) {
	        return new ApplyRegistryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.rejected = source["rejected"];
	    }
	}
	export class InstallRequest {
	    host: string;
	    ssh_user: string;
	    ssh_port: number;
	    ssh_password: string;
	    deploy_agent_env: boolean;
	    checkout_agent?: boolean;
	    github_token: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.ssh_user = source["ssh_user"];
	        this.ssh_port = source["ssh_port"];
	        this.ssh_password = source["ssh_password"];
	        this.deploy_agent_env = source["deploy_agent_env"];
	        this.checkout_agent = source["checkout_agent"];
	        this.github_token = source["github_token"];
	    }
	}
	export class LoginResult {
	    user_id: string;
	    name: string;
	    phone: string;
	    role: string;
	    is_admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LoginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.name = source["name"];
	        this.phone = source["phone"];
	        this.role = source["role"];
	        this.is_admin = source["is_admin"];
	    }
	}
	export class ProvisionRequest {
	    dtid: string;
	    device_id: string;
	    serial: string;
	    hw_version: string;
	    overwrite?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProvisionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dtid = source["dtid"];
	        this.device_id = source["device_id"];
	        this.serial = source["serial"];
	        this.hw_version = source["hw_version"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class ProvisionResult {
	    global_device_id: string;
	    device_id: string;
	    dtid: string;
	    dsid: string;
	    serial_number: string;
	    mqtt_username: string;
	    mqtt_password: string;
	
	    static createFrom(source: any = {}) {
	        return new ProvisionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.global_device_id = source["global_device_id"];
	        this.device_id = source["device_id"];
	        this.dtid = source["dtid"];
	        this.dsid = source["dsid"];
	        this.serial_number = source["serial_number"];
	        this.mqtt_username = source["mqtt_username"];
	        this.mqtt_password = source["mqtt_password"];
	    }
	}
	export class SessionInfo {
	    logged_in: boolean;
	    user_name: string;
	    phone: string;
	    role: string;
	    gateway_url: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logged_in = source["logged_in"];
	        this.user_name = source["user_name"];
	        this.phone = source["phone"];
	        this.role = source["role"];
	        this.gateway_url = source["gateway_url"];
	    }
	}
	export class VerifyRequest {
	    device_id: string;
	    mqtt_username: string;
	    mqtt_password: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_id = source["device_id"];
	        this.mqtt_username = source["mqtt_username"];
	        this.mqtt_password = source["mqtt_password"];
	    }
	}
	export class VerifyResult {
	    mqtt: verify.Result;
	    install?: device.InstallResult;
	
	    static createFrom(source: any = {}) {
	        return new VerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mqtt = this.convertValues(source["mqtt"], verify.Result);
	        this.install = this.convertValues(source["install"], device.InstallResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace verify {
	
	export class Result {
	    connected: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.message = source["message"];
	    }
	}

}

