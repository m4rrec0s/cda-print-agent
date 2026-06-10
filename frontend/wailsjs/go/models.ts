export namespace main {
	
	export class AgentConfig {
	    wsUrl: string;
	    apiUrl: string;
	    agentKey: string;
	    hotFolderPath: string;
	    deviceName: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wsUrl = source["wsUrl"];
	        this.apiUrl = source["apiUrl"];
	        this.agentKey = source["agentKey"];
	        this.hotFolderPath = source["hotFolderPath"];
	        this.deviceName = source["deviceName"];
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

