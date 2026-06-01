export namespace main {
	
	export class AgentConfig {
	    wsUrl: string;
	    apiUrl: string;
	    agentKey: string;
	    hotFolderPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wsUrl = source["wsUrl"];
	        this.apiUrl = source["apiUrl"];
	        this.agentKey = source["agentKey"];
	        this.hotFolderPath = source["hotFolderPath"];
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
