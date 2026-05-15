export namespace capture {
	
	export class BodyView {
	    text: string;
	    base64: string;
	    size: number;
	    truncated: boolean;
	    contentType: string;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new BodyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.base64 = source["base64"];
	        this.size = source["size"];
	        this.truncated = source["truncated"];
	        this.contentType = source["contentType"];
	        this.encoding = source["encoding"];
	    }
	}
	export class CertificateInfo {
	    serverName: string;
	    issuer: string;
	    subject: string;
	    notBefore: string;
	    notAfter: string;
	
	    static createFrom(source: any = {}) {
	        return new CertificateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverName = source["serverName"];
	        this.issuer = source["issuer"];
	        this.subject = source["subject"];
	        this.notBefore = source["notBefore"];
	        this.notAfter = source["notAfter"];
	    }
	}
	export class Timing {
	    startedAt: string;
	    finishedAt: string;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Timing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class SessionDetail {
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
	    requestHeaders: Record<string, Array<string>>;
	    responseHeaders: Record<string, Array<string>>;
	    requestBody: BodyView;
	    responseBody: BodyView;
	    timing: Timing;
	    certificate: CertificateInfo;
	
	    static createFrom(source: any = {}) {
	        return new SessionDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.startedAt = source["startedAt"];
	        this.durationMs = source["durationMs"];
	        this.source = source["source"];
	        this.method = source["method"];
	        this.scheme = source["scheme"];
	        this.host = source["host"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.statusCode = source["statusCode"];
	        this.status = source["status"];
	        this.protocol = source["protocol"];
	        this.contentType = source["contentType"];
	        this.responseSize = source["responseSize"];
	        this.requestSize = source["requestSize"];
	        this.interceptedTls = source["interceptedTls"];
	        this.tunnelOnly = source["tunnelOnly"];
	        this.error = source["error"];
	        this.rule = source["rule"];
	        this.responsePreview = source["responsePreview"];
	        this.requestHeaders = source["requestHeaders"];
	        this.responseHeaders = source["responseHeaders"];
	        this.requestBody = this.convertValues(source["requestBody"], BodyView);
	        this.responseBody = this.convertValues(source["responseBody"], BodyView);
	        this.timing = this.convertValues(source["timing"], Timing);
	        this.certificate = this.convertValues(source["certificate"], CertificateInfo);
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
	export class SessionSummary {
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
	
	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.startedAt = source["startedAt"];
	        this.durationMs = source["durationMs"];
	        this.source = source["source"];
	        this.method = source["method"];
	        this.scheme = source["scheme"];
	        this.host = source["host"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.statusCode = source["statusCode"];
	        this.status = source["status"];
	        this.protocol = source["protocol"];
	        this.contentType = source["contentType"];
	        this.responseSize = source["responseSize"];
	        this.requestSize = source["requestSize"];
	        this.interceptedTls = source["interceptedTls"];
	        this.tunnelOnly = source["tunnelOnly"];
	        this.error = source["error"];
	        this.rule = source["rule"];
	        this.responsePreview = source["responsePreview"];
	    }
	}

}

export namespace main {
	
	export class Status {
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
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.proxyRunning = source["proxyRunning"];
	        this.interceptHttps = source["interceptHttps"];
	        this.proxyPort = source["proxyPort"];
	        this.localProxyAddress = source["localProxyAddress"];
	        this.lanProxyAddresses = source["lanProxyAddresses"];
	        this.certificateUrl = source["certificateUrl"];
	        this.certificatePath = source["certificatePath"];
	        this.certificateSubject = source["certificateSubject"];
	        this.certificateExpiresAt = source["certificateExpiresAt"];
	        this.sessionCount = source["sessionCount"];
	        this.systemProxyHint = source["systemProxyHint"];
	        this.tunStatus = source["tunStatus"];
	        this.upstreamProxy = source["upstreamProxy"];
	    }
	}

}

