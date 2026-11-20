import { POLLING_INTERVAL_MS, RUNNING_DEPLOYMENT_POLLING_INTERVAL_MS } from '#lib/config.ts';
import fetcher, {
	type FetchOptions,
	type FetchService,
	type QueryResult
} from '#lib/fetcher/index.ts';
import type { Paginated } from '#lib/pagination.ts';
import type { TargetStatus } from '#lib/resources/targets.ts';
import type { ByUserData } from '#lib/resources/users.ts';

export enum DeploymentStatus {
	Pending = 0,
	Running = 1,
	Failed = 2,
	Succeeded = 3
}

export type SourceData =
	| {
			discriminator: 'git';
			data: { branch: string; hash: string };
	  }
	| {
			discriminator: 'archive' | 'raw';
			data: string;
	  };

export type SourceDataDiscriminator = SourceData['discriminator'];

export type Entrypoint = {
	name: string;
	router: string;
	port: number;
	is_custom: boolean;
	subdomain?: string;
	url?: string;
	published_port?: number;
};

export type Service = {
	name: string;
	image: string;
	entrypoints: Entrypoint[];
};

export type State = {
	status: DeploymentStatus;
	error_code?: string;
	started_at?: string;
	finished_at?: string;
};

export type StateWithServices = State & {
	services: Service[];
};

export type Environment = 'production' | 'staging';

export type TargetSummary = {
	id: string;
	name?: string;
	url?: string;
	status?: TargetStatus;
};

export type Deployment = {
	app_id: string;
	deployment_number: number;
	environment: Environment;
	target: TargetSummary;
	source: SourceData;
	state: State;
	requested_at: string;
	requested_by: ByUserData;
};

export type DeploymentDetail = Omit<Deployment, 'state'> & {
	state: StateWithServices;
};

export type QueueDeployment =
	| FormData
	| ({
			environment: Environment;
	  } & ({ raw: string } | { git: { branch: string; hash?: string } }));

export type QueryDeploymentsFilters = {
	page?: number;
	environment?: Environment;
	status?: DeploymentStatus;
	/** A deployment number (with or without #), or part of a Git branch or commit. */
	search?: string;
	per_page?: number;
};

export interface DeploymentsService {
	queue(appid: string, data: QueueDeployment): Promise<Deployment>;
	redeploy(appid: string, number: number): Promise<Deployment>;
	promote(appid: string, number: number): Promise<Deployment>;
	/** Cancels a pending or running deployment (seelf#7); answers with it as it is now. */
	cancel(appid: string, number: number): Promise<DeploymentDetail>;
	/** Where the live log of a deployment is streamed (server-sent events). */
	logsStreamUrl(appid: string, number: number): string;
	/** With `poll`, refreshed like a running deployment so new deploys show up on their own. */
	queryAllByApp(
		id: string,
		filters?: QueryDeploymentsFilters,
		poll?: boolean
	): QueryResult<Paginated<Deployment>>;
	queryLogs(appid: string, number: number, poll?: boolean): QueryResult<string>;
	queryByAppAndNumber(appid: string, number: number, poll?: boolean): QueryResult<DeploymentDetail>;
	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentDetail>;
}

type Options = {
	pollingInterval: number;
	runningDeploymentsPollingInterval: number;
};

class RemoteDeploymentsService implements DeploymentsService {
	constructor(
		private readonly _fetcher: FetchService,
		private readonly _options: Options
	) {}

	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentDetail> {
		return this._fetcher.get(`/api/v1/apps/${appid}/deployments/${number}`, options);
	}

	queryLogs(appid: string, number: number, poll?: boolean): QueryResult<string> {
		return this._fetcher.query(`/api/v1/apps/${appid}/deployments/${number}/logs`, {
			refreshInterval: poll ? this._options.runningDeploymentsPollingInterval : undefined,
			cache: 'no-store' // The log grows while the deployment runs: always ask the server for the latest content
		});
	}

	queryByAppAndNumber(
		appid: string,
		number: number,
		poll?: boolean
	): QueryResult<DeploymentDetail> {
		return this._fetcher.query(`/api/v1/apps/${appid}/deployments/${number}`, {
			refreshInterval: poll ? this._options.runningDeploymentsPollingInterval : undefined
		});
	}

	queue(appid: string, data: QueueDeployment): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments`, data, {
			invalidate: [`/api/v1/apps/${appid}`, '/api/v1/apps']
		});
	}

	redeploy(appid: string, number: number): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments/${number}/redeploy`, undefined, {
			invalidate: [`/api/v1/apps/${appid}`, `/api/v1/apps/${appid}/deployments`, '/api/v1/apps']
		});
	}

	promote(appid: string, number: number): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments/${number}/promote`, undefined, {
			invalidate: [`/api/v1/apps/${appid}`, `/api/v1/apps/${appid}/deployments`, '/api/v1/apps']
		});
	}

	cancel(appid: string, number: number): Promise<DeploymentDetail> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments/${number}/cancel`, undefined, {
			invalidate: [
				`/api/v1/apps/${appid}`,
				`/api/v1/apps/${appid}/deployments`,
				`/api/v1/apps/${appid}/deployments/${number}`,
				'/api/v1/apps'
			]
		});
	}

	logsStreamUrl(appid: string, number: number): string {
		return `/api/v1/apps/${appid}/deployments/${number}/logs/stream`;
	}

	queryAllByApp(
		id: string,
		filters?: QueryDeploymentsFilters,
		poll?: boolean
	): QueryResult<Paginated<Deployment>> {
		return this._fetcher.query(`/api/v1/apps/${id}/deployments`, {
			params: filters,
			refreshInterval: poll ? this._options.runningDeploymentsPollingInterval : undefined
		});
	}
}

const service: DeploymentsService = new RemoteDeploymentsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS,
	runningDeploymentsPollingInterval: RUNNING_DEPLOYMENT_POLLING_INTERVAL_MS
});

export default service;
