import fetcher, { type FetchOptions, type QueryResult } from '#lib/fetcher/index.ts';
import type { Paginated } from '#lib/pagination.ts';
import type { Deployment, Environment } from '#lib/resources/deployments.ts';

/** Polling fallback: live hints come over GET /api/v1/events, but some changes send none. */
export const DASHBOARD_FALLBACK_POLL_MS = 30_000;

export type DashboardRange = 7 | 30 | 90;
export type AppsSort = 'deploys' | 'failures' | 'recent';

export type Period = {
	deployments: number;
	succeeded: number;
	failed: number;
	other: number;
	/** 0..1 over finished deployments */
	success_rate: number | null;
	/** seconds */
	median_duration: number | null;
	max_duration: number | null;
	median_recovery: number | null;
	recoveries: number;
};

export type Day = {
	/** YYYY-MM-DD in the viewer's time zone */
	date: string;
	succeeded: number;
	failed: number;
	other: number;
	median_duration: number | null;
	max_duration: number | null;
};

export type Summary = {
	days: number;
	from: string;
	until: string;
	apps: number;
	current: Period;
	previous: Period;
	series: Day[];
	by_environment: {
		environment: Environment;
		deployments: number;
		succeeded: number;
		failed: number;
	}[];
	errors: { code: string; count: number }[];
	recovery: { environment: Environment };
};

export type AppStat = {
	id: string;
	name: string;
	deployments: number;
	succeeded: number;
	failed: number;
	success_rate: number | null;
	last_deployment: string | null;
};

export type Activity = {
	data: Deployment[];
	next_cursor: string | null;
};

export type DashboardFilters = {
	range: DashboardRange;
	environment?: Environment;
	/** One application's deployments only (the summary on its page). */
	app?: string;
	/** Deployments to one target only (the summary on its page). */
	target?: string;
};

export type AppsFilters = DashboardFilters & {
	sort: AppsSort;
	page: number;
	perPage: number;
};

export type ActivityFilters = {
	/** Only deployments of the last 7, 30 or 90 days (the summary's period) */
	range?: DashboardRange;
	environment?: Environment;
	status?: number;
	app?: string;
	cursor?: string;
	limit: number;
};

/** The viewer's offset from UTC in minutes, east positive (what the API expects as `tz`). */
export const viewerOffset = () => -new Date().getTimezoneOffset();

const base = ({ range, environment, app, target }: DashboardFilters) => ({
	range,
	tz: viewerOffset(),
	...(environment ? { environment } : {}),
	...(app ? { app } : {}),
	...(target ? { target } : {})
});

const appsParams = (f: AppsFilters) => ({
	...base(f),
	sort: f.sort,
	page: f.page,
	per_page: f.perPage
});

const activityParams = (f: ActivityFilters) => ({
	limit: f.limit,
	...(f.range ? { range: f.range, tz: viewerOffset() } : {}),
	...(f.environment ? { environment: f.environment } : {}),
	...(f.status !== undefined ? { status: f.status } : {}),
	...(f.app ? { app: f.app } : {}),
	...(f.cursor ? { cursor: f.cursor } : {})
});

export interface DashboardService {
	querySummary(filters: DashboardFilters): QueryResult<Summary>;
	fetchSummary(filters: DashboardFilters, options?: FetchOptions): Promise<Summary>;
	queryApps(filters: AppsFilters): QueryResult<Paginated<AppStat>>;
	fetchApps(filters: AppsFilters, options?: FetchOptions): Promise<Paginated<AppStat>>;
	fetchActivity(filters: ActivityFilters, options?: FetchOptions): Promise<Activity>;
}

const service: DashboardService = {
	querySummary: (filters) =>
		fetcher.query('/api/v1/dashboard/summary', {
			refreshInterval: DASHBOARD_FALLBACK_POLL_MS,
			params: base(filters)
		}),
	fetchSummary: (filters, options) =>
		fetcher.get('/api/v1/dashboard/summary', { ...options, params: base(filters) }),
	queryApps: (filters) =>
		fetcher.query('/api/v1/dashboard/apps', {
			refreshInterval: DASHBOARD_FALLBACK_POLL_MS,
			params: appsParams(filters)
		}),
	fetchApps: (filters, options) =>
		fetcher.get('/api/v1/dashboard/apps', { ...options, params: appsParams(filters) }),
	fetchActivity: (filters, options) =>
		fetcher.get('/api/v1/dashboard/activity', {
			cache: 'no-store',
			...options,
			params: activityParams(filters)
		})
};

export default service;
