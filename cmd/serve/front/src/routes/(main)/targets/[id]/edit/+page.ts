import service from '#lib/resources/targets.ts';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		const target = await service.fetchById(params.id, { fetch, depends });

		return {
			target
		};
	} catch {
		error(404);
	}
};
