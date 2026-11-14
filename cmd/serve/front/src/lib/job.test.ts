import { describe, expect, it } from 'vitest';
import { deploymentEnvironment, jobState, jobSubject, secondsUntil } from '#lib/job.ts';
import { JobPolicy, type Job } from '#lib/resources/jobs.ts';

const names = {
	app: (id: string) => ({ app1: 'my-api' })[id],
	target: (id: string) => ({ t1: 'hetzner-1' })[id]
};

function job(over: Partial<Job>): Job {
	return {
		id: 'j',
		resource_id: 'r',
		group: 'g',
		message_name: 'x',
		message_data: '{}',
		queued_at: '2026-11-14T10:00:00Z',
		not_before: '2026-11-14T10:00:00Z',
		policy: 0,
		retrieved: false,
		...over
	};
}

describe('deploymentEnvironment', () => {
	it('reads the environment from the deployment group', () => {
		expect(deploymentEnvironment('deployment.deployment.deploy.my-api-staging-app1', 'APP1')).toBe(
			'staging'
		);
		expect(deploymentEnvironment('deployment.deployment.deploy.api-production-x', 'x')).toBe(
			'production'
		);
	});

	it('returns nothing for other groups', () => {
		expect(deploymentEnvironment('j1', 'app1')).toBeUndefined();
		expect(
			deploymentEnvironment('deployment.deployment.deploy.api-production-x', 'y')
		).toBeUndefined();
	});
});

describe('jobSubject', () => {
	it('names a deployment after its app and number', () => {
		expect(
			jobSubject(
				job({
					message_name: 'deployment.command.deploy',
					message_data: '{"app_id":"app1","deployment_number":5}',
					group: 'deployment.deployment.deploy.my-api-staging-app1'
				}),
				names
			)
		).toEqual({
			kind: 'deployment',
			label: 'my-api #5',
			environment: 'staging',
			href: '/apps/app1/deployments/5'
		});
	});

	it('names a target job after the target', () => {
		expect(
			jobSubject(
				job({ message_name: 'deployment.command.configure_target', message_data: '{"id":"t1"}' }),
				names
			)
		).toEqual({ kind: 'target', label: 'hetzner-1', href: '/targets/t1' });
	});

	it('falls back to the resource id', () => {
		expect(jobSubject(job({ message_data: 'nope', resource_id: 'res' }), names)).toEqual({
			kind: 'unknown',
			label: 'res'
		});
	});
});

describe('jobState', () => {
	it('follows the server fields', () => {
		expect(jobState(job({ retrieved: true, error_code: 'e' }), []).state).toBe('running');
		expect(jobState(job({ error_code: 'e' }), []).state).toBe('failed');
		expect(jobState(job({}), []).state).toBe('pending');
	});

	it('waits behind a running job of its group', () => {
		const running = job({ id: 'a', retrieved: true });
		const next = job({ id: 'b' });
		expect(jobState(next, [running, next])).toEqual({ state: 'waiting', after: running });
	});

	it('waits for earlier jobs on the same resource when its policy says so', () => {
		const first = job({ id: 'a', group: 'a', resource_id: 'r' });
		const next = job({
			id: 'b',
			group: 'b',
			resource_id: 'r',
			policy: JobPolicy.WaitForOthersResourceID,
			queued_at: '2026-11-14T10:01:00Z'
		});
		expect(jobState(next, [first, next])).toEqual({ state: 'waiting', after: first });
	});
});

describe('secondsUntil', () => {
	it('never goes below zero', () => {
		const now = Date.parse('2026-11-14T10:00:00Z');
		expect(secondsUntil('2026-11-14T10:00:12Z', now)).toBe(12);
		expect(secondsUntil('2026-11-14T09:00:00Z', now)).toBe(0);
	});
});
