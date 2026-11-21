import { describe, expect, it } from 'vitest';
import {
	domainChips,
	isUnder,
	parseHostname,
	recordFor,
	unicodeOf,
	worstTone
} from './hostname.ts';

describe('parseHostname', () => {
	it('normalizes valid hostnames', () => {
		expect(parseHostname('  Shop.Example.COM. ')).toEqual({ value: 'shop.example.com' });
		expect(parseHostname('bücher.example')).toEqual({ value: 'xn--bcher-kva.example' });
	});

	it('says what is wrong', () => {
		expect(parseHostname('')).toEqual({ error: 'hostname.required' });
		expect(parseHostname('https://shop.example.com')).toEqual({ error: 'hostname.not_a_name' });
		expect(parseHostname('shop.example.com/path')).toEqual({ error: 'hostname.not_a_name' });
		expect(parseHostname('shop.example.com:8080')).toEqual({ error: 'hostname.not_a_name' });
		expect(parseHostname('*.example.com')).toEqual({ error: 'hostname.wildcard' });
		expect(parseHostname('203.0.113.10')).toEqual({ error: 'hostname.ip' });
		expect(parseHostname('localhost')).toEqual({ error: 'invalid_hostname' });
		expect(parseHostname('-shop.example.com')).toEqual({ error: 'invalid_hostname' });
		expect(parseHostname('shop_1.example.com')).toEqual({ error: 'invalid_hostname' });
		expect(parseHostname('a'.repeat(64) + '.example.com')).toEqual({ error: 'invalid_hostname' });
	});
});

describe('helpers', () => {
	it('tells whether a hostname is under a host', () => {
		expect(isUnder('shop.apps.example.com', 'apps.example.com')).toBe(true);
		expect(isUnder('apps.example.com', 'apps.example.com')).toBe(true);
		expect(isUnder('shop.ps.example.com', 'apps.example.com')).toBe(false);
		expect(isUnder('shop.example.com', undefined)).toBe(false);
	});

	it('shows internationalized names', () => {
		expect(unicodeOf('xn--bcher-kva.example')).toBe('bücher.example');
		expect(unicodeOf('shop.example.com')).toBe('shop.example.com');
	});

	it('gives a CNAME for subdomains and addresses for bare domains', () => {
		expect(recordFor('shop.example.com', 'shop.apps.example.net', ['203.0.113.10'])).toEqual({
			hostname: 'shop.example.com',
			type: 'CNAME',
			values: ['shop.apps.example.net']
		});
		expect(
			recordFor('example.com', 'shop.apps.example.net', ['203.0.113.10', '2001:db8::10'])
		).toEqual({
			hostname: 'example.com',
			type: 'A / AAAA',
			values: ['203.0.113.10', '2001:db8::10']
		});
	});

	it('maps checks to chips', () => {
		const base = {
			service: 'web',
			hostname: 'shop.example.com',
			unicode: 'shop.example.com',
			addresses: [],
			caa: 'caa_ok' as const
		};

		expect(domainChips(undefined, true)).toEqual([
			{ key: 'domains.status.unchecked', tone: 'muted' }
		]);
		const ok = domainChips({ ...base, dns: 'points_here', certificate: 'certificate_ok' }, true);
		expect(ok.map((c) => c.key)).toEqual([
			'domains.status.points_here',
			'domains.status.certificate_ok'
		]);
		expect(worstTone(ok)).toBe('success');

		const pending = domainChips(
			{ ...base, dns: 'points_here', certificate: 'default_certificate' },
			true
		);
		expect(worstTone(pending)).toBe('warning');

		const blocked = domainChips({ ...base, dns: 'points_here', caa: 'caa_blocks' }, false);
		expect(blocked.map((c) => c.key)).toEqual([
			'domains.status.points_here',
			'domains.status.caa_blocks'
		]);
		expect(worstTone(blocked)).toBe('danger');

		expect(domainChips({ ...base, dns: 'behind_cloudflare' }, true)[0].tone).toBe('muted');
	});
});
