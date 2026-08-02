export namespace main {
	
	export class AgentConfig {
	    wsUrl: string;
	    apiUrl: string;
	    agentKey: string;
	    hotFolderPath: string;
	    deviceId?: string;
	    deviceName?: string;
	    printerPhoto?: string;
	    printerLetter?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wsUrl = source["wsUrl"];
	        this.apiUrl = source["apiUrl"];
	        this.agentKey = source["agentKey"];
	        this.hotFolderPath = source["hotFolderPath"];
	        this.deviceId = source["deviceId"];
	        this.deviceName = source["deviceName"];
	        this.printerPhoto = source["printerPhoto"];
	        this.printerLetter = source["printerLetter"];
	    }
	}
	export class DashboardJobFile {
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardJobFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class DashboardJob {
	    id: string;
	    customer: string;
	    status: string;
	    createdAt: string;
	    updatedAt: string;
	    printerRole: string;
	    files: DashboardJobFile[];
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customer = source["customer"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.printerRole = source["printerRole"];
	        this.files = this.convertValues(source["files"], DashboardJobFile);
	        this.lastError = source["lastError"];
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
	
	export class DashboardSnapshot {
	    status: string;
	    photo: string;
	    letter: string;
	    today: number;
	    printed: number;
	    queued: number;
	    failed: number;
	    jobs: DashboardJob[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.photo = source["photo"];
	        this.letter = source["letter"];
	        this.today = source["today"];
	        this.printed = source["printed"];
	        this.queued = source["queued"];
	        this.failed = source["failed"];
	        this.jobs = this.convertValues(source["jobs"], DashboardJob);
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
	export class PaperSizeInfo {
	    name: string;
	    kind: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new PaperSizeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class SavedArtInfo {
	    name: string;
	    path: string;
	    sizeBytes: number;
	    modifiedAt: string;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SavedArtInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.modifiedAt = source["modifiedAt"];
	        this.isDir = source["isDir"];
	    }
	}
	export class VersionInfo {
	    version: string;
	    downloadUrl: string;
	    releaseNotes: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNotes = source["releaseNotes"];
	    }
	}

}

