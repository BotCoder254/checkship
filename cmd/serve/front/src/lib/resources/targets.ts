import { POLLING_INTERVAL_MS } from '#lib/config.ts';
import fetcher, {
	type FetchOptions,
	type FetchService,
	type QueryResult
} from '#lib/fetcher/index.ts';
import type { ByUserData } from '#lib/resources/users.ts';

export enum TargetStatus {
	Configuring = 0,
	Failed = 1,
	Ready = 2
}

export type TargetState = {
	status: TargetStatus;
	error_code?: string;
	last_ready_version?: string;
};

export type ProviderConfigData = {
	kind: 'docker';
	data: {
		host?: string;
		user?: string;
		port?: number;
		private_key?: string;
	};
};

export type ProviderTypes = ProviderConfigData['kind'];

export type Target = {
	id: string;
	name: string;
	url?: string;
	provider: ProviderConfigData;
	state: TargetState;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
};

export type CreateTarget = {
	name: string;
	url?: string;
	docker?: {
		host?: string;
		user?: string;
		port?: number;
		private_key?: string;
	};
};

export type UpdateTarget = {
	name?: string;
	url: Patch<string>;
	docker?: {
		host?: string;
		user?: string;
		port?: number;
		private_key: Patch<string>;
	};
};

/** Connect Cloudflare (DESIGN.md §11.7): what a target is connected to. The token is never returned. */
export type CloudflareMode = 'tunnel' | 'dns';

export type CloudflareStatus = {
	connected: boolean;
	account_name?: string;
	/** In every Cloudflare dashboard URL; not a secret. */
	account_id?: string;
	tunnel_allowed: boolean;
	mode?: CloudflareMode;
	zones: string[];
	tunnel?: { name: string; status: string; connections: number };
	records: { hostname: string; type: string; content: string }[];
	connected_at?: string;
};

export type CloudflareTaskKey = 'tunnel' | 'routes' | 'records' | 'connector' | 'cleanup';

export type CloudflareSetupResult = {
	tasks: { key: CloudflareTaskKey; detail: string }[];
	/** Records checkship didn't create, left alone unless replaced. */
	conflicts: {
		hostname: string;
		type: string;
		content: string;
		zone: string;
		/** record: someone else's is in the way; dns_edit: the token can't write DNS, add it by hand. */
		reason: 'record' | 'dns_edit';
		/** The record to put instead, as Cloudflare's Add record form asks for it. */
		want: { type: string; name: string; content: string; proxied: boolean };
	}[];
	/** Domains in no zone of the token's account: to set up by hand. */
	outside: string[];
	status: CloudflareStatus;
};

export type GetTargetsFilters = {
	active_only?: boolean;
};

export interface TargetsService {
	create(payload: CreateTarget): Promise<Target>;
	update(id: string, payload: UpdateTarget): Promise<Target>;
	reconfigure(id: string): Promise<void>;
	fetchAll(filters?: GetTargetsFilters, options?: FetchOptions): Promise<Target[]>;
	fetchById(id: string, options?: FetchOptions): Promise<Target>;
	queryAll(): QueryResult<Target[]>;
	delete(id: string): Promise<void>;
	cloudflare(id: string): Promise<CloudflareStatus>;
	connectCloudflare(id: string, token: string): Promise<CloudflareStatus>;
	setupCloudflare(
		id: string,
		payload: { mode?: CloudflareMode; replace?: string[] }
	): Promise<CloudflareSetupResult>;
	disconnectCloudflare(id: string): Promise<void>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteTargetsService implements TargetsService {
	constructor(
		private readonly _fetcher: FetchService,
		private readonly _options: Options
	) {}

	create(payload: CreateTarget): Promise<Target> {
		return this._fetcher.post('/api/v1/targets', payload);
	}

	update(id: string, payload: UpdateTarget): Promise<Target> {
		return this._fetcher.patch(`/api/v1/targets/${id}`, payload, {
			invalidate: ['/api/v1/targets']
		});
	}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/targets/${id}`, {
			invalidate: ['/api/v1/targets']
		});
	}

	reconfigure(id: string): Promise<void> {
		return this._fetcher.post(`/api/v1/targets/${id}/reconfigure`, undefined, {
			invalidate: [`/api/v1/targets/${id}`, '/api/v1/targets']
		});
	}

	cloudflare(id: string): Promise<CloudflareStatus> {
		return this._fetcher.get(`/api/v1/targets/${id}/cloudflare`, { cache: 'no-store' });
	}

	connectCloudflare(id: string, token: string): Promise<CloudflareStatus> {
		return this._fetcher.put(`/api/v1/targets/${id}/cloudflare`, { token });
	}

	setupCloudflare(
		id: string,
		payload: { mode?: CloudflareMode; replace?: string[] }
	): Promise<CloudflareSetupResult> {
		return this._fetcher.post(`/api/v1/targets/${id}/cloudflare/setup`, payload);
	}

	disconnectCloudflare(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/targets/${id}/cloudflare`);
	}

	fetchAll(filters?: GetTargetsFilters, options?: FetchOptions): Promise<Target[]> {
		return this._fetcher.get('/api/v1/targets', {
			...options,
			params: filters
		});
	}

	fetchById(id: string, options?: FetchOptions): Promise<Target> {
		return this._fetcher.get(`/api/v1/targets/${id}`, options);
	}

	queryAll(): QueryResult<Target[]> {
		return this._fetcher.query('/api/v1/targets', {
			refreshInterval: this._options.pollingInterval
		});
	}
}

const service: TargetsService = new RemoteTargetsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;
