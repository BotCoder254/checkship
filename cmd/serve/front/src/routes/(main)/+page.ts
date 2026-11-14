import dashboard from '#lib/resources/dashboard.ts';

// Warm the summary for the default filters so the KPI tiles render with the page.
export const load = async ({ fetch, depends }) => {
	const summary = await dashboard
		.fetchSummary({ range: 30 }, { fetch, depends })
		.catch(() => undefined);

	return { summary };
};
