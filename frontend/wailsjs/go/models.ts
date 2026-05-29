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
	    }
	}

}

export namespace device {
	
	export class InstallResult {
	    agent_active: boolean;
	    agent_checked_out: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent_active = source["agent_active"];
	        this.agent_checked_out = source["agent_checked_out"];
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

export namespace main {
	
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
	    serial: string;
	    hw_version: string;
	    overwrite?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProvisionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial = source["serial"];
	        this.hw_version = source["hw_version"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class ProvisionResult {
	    device_id: string;
	    serial_number: string;
	    mqtt_username: string;
	    mqtt_password: string;
	
	    static createFrom(source: any = {}) {
	        return new ProvisionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_id = source["device_id"];
	        this.serial_number = source["serial_number"];
	        this.mqtt_username = source["mqtt_username"];
	        this.mqtt_password = source["mqtt_password"];
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

