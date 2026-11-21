import fetcher, {
	type FetchOptions,
	type FetchService,
	type QueryResult
} from '#lib/fetcher/index.ts';
import { POLLING_INTERVAL_MS } from '#lib/config.ts';
import type { ByUserData } from '#lib/resources/users.ts';
import type { Deployment, DeploymentDetail } from '#lib/resources/deployments.ts';

export type App = {
	id: string;
	name: string;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
	latest_deployments: LatestDeployments<Deployment>;
	production_target: TargetSummary;
	staging_target: TargetSummary;
};

export type LatestDeployments<T> = {
	production?: T;
	staging?: T;
};

export type EnvironmentVariablesPerService = Record<string, Record<string, string>>;
export type VersionControl = { url: string; token?: string };

export type TargetSummary = {
	id: string;
	name: string;
	url?: string;
};

export type AppDetail = {
	id: string;
	name: string;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
	latest_deployments: LatestDeployments<DeploymentDetail>;
	version_control?: VersionControl;
	production: EnvironmentConfig;
	staging: EnvironmentConfig;
};

/** Custom domains per compose service (seelf#5); the first of the main service is the primary. */
export type DomainsPerService = Record<string, string[]>;

export type EnvironmentConfig = {
	target: TargetSummary;
	vars?: EnvironmentVariablesPerService;
	domains?: DomainsPerService;
};

export type CreateAppDataEnvironmentConfig = {
	target: string;
	vars?: EnvironmentVariablesPerService;
	/** Omitted: the environment keeps its domains. {} removes them. */
	domains?: DomainsPerService;
};

export type DnsStatus =
	'points_here' | 'behind_cloudflare' | 'points_elsewhere' | 'no_record' | 'unknown';
export type CaaStatus = 'caa_ok' | 'caa_blocks' | 'caa_unknown';
export type CertificateStatus = 'certificate_ok' | 'default_certificate' | 'unreachable';

export type DomainStatus = {
	service: string;
	hostname: string;
	unicode: string;
	dns: DnsStatus;
	addresses: string[];
	caa: CaaStatus;
	caa_issuers?: string[];
	/** Its zone's nameservers are Cloudflare's: Connect Cloudflare can set it up. */
	cloudflare_dns?: boolean;
	certificate?: CertificateStatus;
	issuer?: string;
	expires_at?: string;
};

export type DomainsCheck = {
	environment: string;
	https: boolean;
	/** The environment's default address on its target, and what it resolves to. */
	expected: { host: string; addresses: string[] };
	domains: DomainStatus[];
};

export type CreateApp = {
	name: string;
	version_control?: VersionControl;
	production: CreateAppDataEnvironmentConfig;
	staging: CreateAppDataEnvironmentConfig;
};

export type UpdateApp = {
	version_control: Patch<{
		url: string;
		token: Patch<string>;
	}>;
	production: Maybe<CreateAppDataEnvironmentConfig>;
	staging: Maybe<CreateAppDataEnvironmentConfig>;
	/** When a target changes, copy the volume data to the new one (the server defaults to true). */
	move_volumes?: boolean;
};

export interface AppsService {
	create(payload: CreateApp): Promise<AppDetail>;
	update(id: string, payload: UpdateApp): Promise<AppDetail>;
	/** DNS, CAA and certificate status of an environment's saved custom domains. */
	checkDomains(id: string, environment: string): Promise<DomainsCheck>;
	/** Unless `backupVolumes` is false, each environment's volumes are backed up before removal. */
	delete(id: string, backupVolumes?: boolean): Promise<void>;
	fetchAll(options?: FetchOptions): Promise<App[]>;
	fetchById(id: string, options?: FetchOptions): Promise<AppDetail>;
	queryAll(): QueryResult<App[]>;
	queryById(id: string): QueryResult<AppDetail>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteAppsService implements AppsService {
	constructor(
		private readonly _fetcher: FetchService,
		private readonly _options: Options
	) {}

	create(payload: CreateApp): Promise<AppDetail> {
		return this._fetcher.post('/api/v1/apps', payload);
	}

	update(id: string, payload: UpdateApp): Promise<AppDetail> {
		return this._fetcher.patch(`/api/v1/apps/${id}`, payload);
	}

	delete(id: string, backupVolumes = true): Promise<void> {
		const query = backupVolumes ? '' : '?backup_volumes=false';
		return this._fetcher.delete(`/api/v1/apps/${id}${query}`, {
			invalidate: ['/api/v1/apps']
		});
	}

	queryAll(): QueryResult<App[]> {
		return this._fetcher.query('/api/v1/apps', { refreshInterval: this._options.pollingInterval });
	}

	checkDomains(id: string, environment: string): Promise<DomainsCheck> {
		return this._fetcher.get(`/api/v1/apps/${id}/domains/check`, {
			params: { environment },
			cache: 'no-store'
		});
	}

	queryById(id: string): QueryResult<AppDetail> {
		return this._fetcher.query(`/api/v1/apps/${id}`, {
			refreshInterval: this._options.pollingInterval
		});
	}

	fetchAll(options?: FetchOptions): Promise<App[]> {
		return this._fetcher.get('/api/v1/apps', options);
	}

	fetchById(id: string, options?: FetchOptions): Promise<AppDetail> {
		return this._fetcher.get(`/api/v1/apps/${id}`, options);
	}
}

const service: AppsService = new RemoteAppsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;

export type ServiceVariables = {
	name: string;
	values: string;
};

const regexEnvKeyValue = /^\s*(?<key>[^=]+)\s*=\s*(?<value>.*)\s*$/gm;

/**
 * Transform a list of service variables into a record.
 */
export function toServiceVariablesRecord(
	services: ServiceVariables[]
): EnvironmentVariablesPerService {
	return services.reduce<EnvironmentVariablesPerService>(
		(result, service) => ({
			...result,
			[service.name]: parseEnv(service.values)
		}),
		{}
	);
}

export function fromServiceVariablesRecord(
	values?: EnvironmentVariablesPerService
): ServiceVariables[] {
	return Object.entries(values ?? {}).map(([name, values]) => ({
		name,
		values: Object.entries(values)
			.map(([key, value]) => `${key}=${value}`)
			.join('\n')
	}));
}

function parseEnv(str: string): Record<string, string> {
	const result: Record<string, string> = {};
	let match: RegExpExecArray | null;

	while ((match = regexEnvKeyValue.exec(str)) !== null) {
		const { key, value } = match.groups!;

		result[key] = value;
	}

	return result;
}
