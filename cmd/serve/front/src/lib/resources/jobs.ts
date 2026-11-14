import { POLLING_INTERVAL_MS } from '#lib/config.ts';
import fetcher, {
	type FetchOptions,
	type FetchService,
	type QueryResult
} from '#lib/fetcher/index.ts';
import type { Paginated } from '#lib/pagination.ts';

export type Job = {
	id: string;
	resource_id: string;
	group: string;
	message_name: string;
	message_data: string;
	queued_at: string;
	not_before: string;
	error_code?: string;
	policy: number;
	retrieved: boolean;
};

export enum JobPolicy {
	PreserveOrder = 1,
	WaitForOthersResourceID = 2,
	Cancellable = 4,
	Mergeable = 8
}

/** Filter on GET /api/v1/jobs (cmd/serve/jobs.go). */
export type JobStatusFilter = 'running' | 'failed' | 'pending';

export type JobsFilters = {
	page: number;
	status?: JobStatusFilter;
};

export interface JobsService {
	delete(id: string): Promise<void>;
	fetchAll(filters: JobsFilters, options?: FetchOptions): Promise<Paginated<Job>>;
	/** Polls every few seconds unless `live` is false. */
	queryAll(filters: JobsFilters, live?: boolean): QueryResult<Paginated<Job>>;
}

const params = ({ page, status }: JobsFilters) => (status ? { page, status } : { page });

type Options = {
	pollingInterval: number;
};

export class RemoteJobsService implements JobsService {
	constructor(
		private readonly _fetcher: FetchService,
		private readonly _options: Options
	) {}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/jobs/${id}`, {
			invalidate: ['/api/v1/jobs']
		});
	}

	queryAll(filters: JobsFilters, live = true): QueryResult<Paginated<Job>> {
		return this._fetcher.query('/api/v1/jobs', {
			refreshInterval: live ? this._options.pollingInterval : undefined,
			params: params(filters)
		});
	}

	fetchAll(filters: JobsFilters, options?: FetchOptions): Promise<Paginated<Job>> {
		return this._fetcher.get('/api/v1/jobs', {
			...options,
			params: params(filters)
		});
	}
}

const service: JobsService = new RemoteJobsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;
