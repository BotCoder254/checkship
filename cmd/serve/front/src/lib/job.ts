import routes from '#lib/path.ts';
import type { Environment } from '#lib/resources/deployments.ts';
import { JobPolicy, type Job } from '#lib/resources/jobs.ts';

/** Where a job stands, as the server keeps it (pkg/bus/sqlite/store.go). */
export type JobState = 'running' | 'failed' | 'waiting' | 'pending';

/** What a job acts on, named for people: "welcome #5" in staging, "hetzner-1"… */
export type JobSubject = {
	label: string;
	environment?: Environment;
	href?: string;
	kind: 'deployment' | 'app' | 'target' | 'unknown';
};

export type Names = {
	app: (id: string) => string | undefined;
	target: (id: string) => string | undefined;
};

const DEPLOY_GROUP = 'deployment.deployment.deploy.';

function payload(job: Pick<Job, 'message_data'>): Record<string, unknown> {
	try {
		const data = JSON.parse(job.message_data);
		return data && typeof data === 'object' ? data : {};
	} catch {
		return {};
	}
}

const str = (value: unknown) => (typeof value === 'string' && value ? value : undefined);
const environmentOf = (value: unknown): Environment | undefined =>
	value === 'production' || value === 'staging' ? value : undefined;

/**
 * The environment of a deployment job, read from its group: the server names it after the
 * compose project, "deployment.deployment.deploy.<app name>-<environment>-<app id>".
 */
export function deploymentEnvironment(group: string, appId: string): Environment | undefined {
	if (!group.startsWith(DEPLOY_GROUP)) return undefined;
	const suffix = `-${appId.toLowerCase()}`;
	const project = group.slice(DEPLOY_GROUP.length);
	if (!project.endsWith(suffix)) return undefined;
	const rest = project.slice(0, -suffix.length);
	return environmentOf(rest.slice(rest.lastIndexOf('-') + 1));
}

export function jobSubject(
	job: Pick<Job, 'message_name' | 'message_data' | 'group' | 'resource_id'>,
	names: Names
): JobSubject {
	const data = payload(job);

	switch (job.message_name) {
		case 'deployment.command.deploy': {
			const appId = str(data.app_id);
			const number = typeof data.deployment_number === 'number' ? data.deployment_number : 0;
			if (!appId || !number) break;
			return {
				kind: 'deployment',
				label: `${names.app(appId) ?? appId} #${number}`,
				environment: deploymentEnvironment(job.group, appId),
				href: routes.deployment(appId, number)
			};
		}
		case 'deployment.command.cleanup_app': {
			const appId = str(data.app_id);
			if (!appId) break;
			return {
				kind: 'app',
				label: names.app(appId) ?? appId,
				environment: environmentOf(data.environment),
				href: names.app(appId) ? routes.app(appId) : undefined
			};
		}
		case 'deployment.command.delete_app': {
			const appId = str(data.id);
			if (!appId) break;
			const name = names.app(appId);
			return { kind: 'app', label: name ?? appId, href: name ? routes.app(appId) : undefined };
		}
		case 'deployment.command.configure_target':
		case 'deployment.command.cleanup_target':
		case 'deployment.command.delete_target': {
			const targetId = str(data.id);
			if (!targetId) break;
			const name = names.target(targetId);
			return {
				kind: 'target',
				label: name ?? targetId,
				href: name ? routes.target(targetId) : undefined
			};
		}
		case 'deployment.command.backup_volumes':
		case 'deployment.command.store_backup':
		case 'deployment.command.restore_volumes':
			// The payload only has the backup or restore id: point at the Backups page.
			return { kind: 'unknown', label: job.resource_id.slice(0, 8), href: routes.backups };
	}

	return { kind: 'unknown', label: job.resource_id };
}

/**
 * The job's state. "Waiting" is only known for jobs on the same page: another job of its group
 * is running, or (for jobs that wait for their resource) another job on the same resource is
 * ahead of it.
 */
export function jobState(job: Job, page: Job[]): { state: JobState; after?: Job } {
	if (job.retrieved) return { state: 'running' };
	if (job.error_code) return { state: 'failed' };

	const sameGroup = page.find(
		(other) => other.id !== job.id && other.retrieved && other.group === job.group
	);
	if (sameGroup) return { state: 'waiting', after: sameGroup };

	if (job.policy & JobPolicy.WaitForOthersResourceID) {
		const ahead = page.find(
			(other) =>
				other.id !== job.id &&
				other.resource_id === job.resource_id &&
				other.queued_at <= job.queued_at
		);
		if (ahead) return { state: 'waiting', after: ahead };
	}

	return { state: 'pending' };
}

export const isCancellable = (job: Pick<Job, 'policy'>) =>
	(job.policy & JobPolicy.Cancellable) !== 0;

/** Seconds until the next try of a failed job, 0 when it is due. */
export function secondsUntil(date: string, now: number): number {
	return Math.max(0, Math.ceil((new Date(date).getTime() - now) / 1000));
}

/** Pretty-printed JSON payload, or the raw text when it isn't JSON. */
export function prettyPayload(data: string): string {
	try {
		return JSON.stringify(JSON.parse(data), null, 2);
	} catch {
		return data;
	}
}
