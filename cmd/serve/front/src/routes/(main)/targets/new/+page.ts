import service from '#lib/resources/targets.ts';
import { isLocal } from '#lib/target.ts';

export const load = async ({ fetch, depends }) => {
	const targets = await service.fetchAll(undefined, { fetch, depends });

	// One target per Docker engine: this machine can only be added once.
	return { localTarget: targets.find((t) => isLocal(t))?.name };
};
