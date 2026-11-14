import service from '#lib/resources/apps.ts';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		const app = await service.fetchById(params.id, { fetch, depends });

		return {
			app
		};
	} catch {
		error(404);
	}
};
