import { describe, expect, it } from 'vitest';
import { DeploymentStatus } from '#lib/resources/deployments.ts';
import {
	DEPLOYMENT_CANCELLED,
	deploymentBadge,
	isCancellable,
	isCancelled,
	isFailure
} from './status.ts';

describe('cancelled deployments', () => {
	const cancelled = { status: DeploymentStatus.Failed, error_code: DEPLOYMENT_CANCELLED };
	const failed = { status: DeploymentStatus.Failed, error_code: 'compose_failed' };

	it('tells a cancellation from a failure', () => {
		expect(isCancelled(cancelled)).toBe(true);
		expect(isFailure(cancelled)).toBe(false);
		expect(isCancelled(failed)).toBe(false);
		expect(isFailure(failed)).toBe(true);
		expect(isCancelled({ status: DeploymentStatus.Succeeded })).toBe(false);
	});

	it('shows a cancelled badge, not a failed one', () => {
		expect(deploymentBadge(cancelled).state).toBe('cancelled');
		expect(deploymentBadge(failed).state).toBe('failed');
	});

	it('only offers cancel while pending or running', () => {
		expect(isCancellable({ status: DeploymentStatus.Pending })).toBe(true);
		expect(isCancellable({ status: DeploymentStatus.Running })).toBe(true);
		expect(isCancellable({ status: DeploymentStatus.Succeeded })).toBe(false);
		expect(isCancellable({ status: DeploymentStatus.Failed })).toBe(false);
	});
});
