export namespace connections {
	
	export class Server {
	    id: string;
	    name: string;
	    url: string;
	    kind: string;
	    username?: string;
	    health: string;
	    // Go type: time
	    lastChecked?: any;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.kind = source["kind"];
	        this.username = source["username"];
	        this.health = source["health"];
	        this.lastChecked = this.convertValues(source["lastChecked"], null);
	        this.active = source["active"];
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

