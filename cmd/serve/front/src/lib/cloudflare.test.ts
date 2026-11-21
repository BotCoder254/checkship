import { describe, expect, it } from 'vitest';
import { cloudflareTokenLink, looksLikeCloudflareToken } from './cloudflare.ts';

describe('cloudflareTokenLink', () => {
	it('should pre-fill the three permissions, every zone and the name', () => {
		const url = new URL(cloudflareTokenLink('checkship · gitdesk'));

		expect(url.origin + url.pathname).toBe('https://dash.cloudflare.com/profile/api-tokens');
		expect(JSON.parse(url.searchParams.get('permissionGroupKeys')!)).toEqual([
			{ key: 'zone', type: 'read' },
			{ key: 'dns', type: 'edit' },
			{ key: 'argotunnel', type: 'edit' }
		]);
		expect(url.searchParams.get('accountId')).toBe('*');
		expect(url.searchParams.get('zoneId')).toBe('all');
		expect(url.searchParams.get('name')).toBe('checkship · gitdesk');
	});
});

describe('looksLikeCloudflareToken', () => {
	it('should accept what a token looks like and refuse the rest', () => {
		expect(looksLikeCloudflareToken('  ' + 'a'.repeat(40) + ' ')).toBe(true);
		expect(looksLikeCloudflareToken('short')).toBe(false);
		expect(looksLikeCloudflareToken('a'.repeat(39) + ' b')).toBe(false);
		expect(looksLikeCloudflareToken('Bearer ' + 'a'.repeat(40))).toBe(false);
	});
});
