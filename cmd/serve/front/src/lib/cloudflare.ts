/**
 * Connect Cloudflare (DESIGN.md §11.7): the link to Cloudflare's token page with the permissions
 * checkship needs already chosen, so the person only clicks "Continue to summary" and "Create
 * Token" there. The token never leaves their browser until they paste it into checkship.
 */

/** The permissions, in Cloudflare's own words, and why each one is needed. */
export const CLOUDFLARE_PERMISSIONS = [
	{ key: 'zone', type: 'read', label: 'Zone · Read' },
	{ key: 'dns', type: 'edit', label: 'DNS · Edit' },
	{ key: 'argotunnel', type: 'edit', label: 'Cloudflare Tunnel · Edit' }
] as const;

const TOKEN_PAGE = 'https://dash.cloudflare.com/profile/api-tokens';

/** Cloudflare's token page, pre-filled: the permissions, every zone of every account, a name. */
export function cloudflareTokenLink(name: string): string {
	const params = [
		[
			'permissionGroupKeys',
			JSON.stringify(CLOUDFLARE_PERMISSIONS.map(({ key, type }) => ({ key, type })))
		],
		['accountId', '*'],
		['zoneId', 'all'],
		['name', name]
	];
	return `${TOKEN_PAGE}?${params.map(([k, v]) => `${k}=${encodeURIComponent(v)}`).join('&')}`;
}

/** What a token looks like (the server checks the same): refused before anything is sent. */
export function looksLikeCloudflareToken(value: string): boolean {
	return /^[A-Za-z0-9_.-]{32,128}$/.test(value.trim());
}
