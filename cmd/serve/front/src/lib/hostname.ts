/**
 * Custom domains in the dashboard (seelf#5): the same hostname rules as the server
 * (internal/deployment/domain/hostname.go), so mistakes show while typing; the server stays the
 * authority. Also what a domain's check means for the person, and the DNS record to create.
 */
import type { DomainStatus } from '#lib/resources/apps.ts';

export type HostnameError =
	| 'hostname.required'
	| 'hostname.not_a_name'
	| 'hostname.wildcard'
	| 'hostname.ip'
	| 'invalid_hostname';

const LABEL = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

/** Normalizes a typed hostname (trimmed, lower case, punycode) or says what is wrong with it. */
export function parseHostname(raw: string): { value: string } | { error: HostnameError } {
	const value = raw.trim().toLowerCase().replace(/\.$/, '');

	if (!value) return { error: 'hostname.required' };
	if (/^[a-z][a-z0-9+.-]*:\/\//.test(value) || /[/:?#@]/.test(value))
		return { error: 'hostname.not_a_name' };
	if (value.includes('*')) return { error: 'hostname.wildcard' };
	if (/^[0-9.]+$/.test(value) || value.includes('[')) return { error: 'hostname.ip' };

	let ascii: string;
	try {
		// The URL parser converts internationalized names to their ASCII (punycode) form.
		ascii = new URL(`http://${value}`).hostname;
	} catch {
		return { error: 'invalid_hostname' };
	}

	const labels = ascii.split('.');
	if (
		ascii.length > 253 ||
		labels.length < 2 ||
		!labels.every((label) => LABEL.test(label)) ||
		/^[0-9]+$/.test(labels[labels.length - 1])
	) {
		return { error: 'invalid_hostname' };
	}

	return { value: ascii };
}

/** Whether a hostname is a given host or one of its subdomains. */
export function isUnder(hostname: string, host: string | undefined): boolean {
	if (!host) return false;
	const base = host.toLowerCase().replace(/\.$/, '');
	return hostname === base || hostname.endsWith(`.${base}`);
}

/** The internationalized form of a punycode hostname, for display (the same when plain ASCII). */
export function unicodeOf(hostname: string): string {
	if (!hostname.split('.').some((label) => label.startsWith('xn--'))) return hostname;
	try {
		return new URL(`http://${hostname}`).hostname === hostname
			? decodePunycode(hostname)
			: hostname;
	} catch {
		return hostname;
	}
}

// Minimal RFC 3492 decoder for display only (no encoding needed: URL does that).
function decodePunycode(hostname: string): string {
	return hostname
		.split('.')
		.map((label) => (label.startsWith('xn--') ? decodeLabel(label.slice(4)) : label))
		.join('.');
}

function decodeLabel(input: string): string {
	const base = 36,
		tMin = 1,
		tMax = 26,
		skew = 38,
		damp = 700;
	let n = 128,
		i = 0,
		bias = 72;
	const output: number[] = [];
	const basic = input.lastIndexOf('-');
	for (let j = 0; j < Math.max(basic, 0); j++) output.push(input.charCodeAt(j));
	const adapt = (delta: number, points: number, first: boolean) => {
		delta = first ? Math.floor(delta / damp) : delta >> 1;
		delta += Math.floor(delta / points);
		let k = 0;
		while (delta > ((base - tMin) * tMax) >> 1) {
			delta = Math.floor(delta / (base - tMin));
			k += base;
		}
		return k + Math.floor(((base - tMin + 1) * delta) / (delta + skew));
	};
	for (let index = basic > 0 ? basic + 1 : 0; index < input.length;) {
		const oldI = i;
		for (let w = 1, k = base; ; k += base) {
			const c = input.charCodeAt(index++);
			const digit = c - 48 < 10 ? c - 22 : c - 65 < 26 ? c - 65 : c - 97 < 26 ? c - 97 : base;
			i += digit * w;
			const t = k <= bias ? tMin : k >= bias + tMax ? tMax : k - bias;
			if (digit < t) break;
			w *= base - t;
		}
		bias = adapt(i - oldI, output.length + 1, oldI === 0);
		n += Math.floor(i / (output.length + 1));
		i %= output.length + 1;
		output.splice(i++, 0, n);
	}
	return String.fromCodePoint(...output);
}

export type ChipTone = 'success' | 'warning' | 'danger' | 'muted';
export type DomainChip = { key: string; tone: ChipTone };

/** What a check says about a domain, as status chips (DESIGN.md §11.4), most important first. */
export function domainChips(status: DomainStatus | undefined, https: boolean): DomainChip[] {
	if (!status) return [{ key: 'domains.status.unchecked', tone: 'muted' }];

	const chips: DomainChip[] = [];
	switch (status.dns) {
		case 'points_here':
			chips.push({ key: 'domains.status.points_here', tone: 'success' });
			break;
		case 'behind_cloudflare':
			chips.push({ key: 'domains.status.behind_cloudflare', tone: 'muted' });
			break;
		case 'points_elsewhere':
			chips.push({ key: 'domains.status.points_elsewhere', tone: 'warning' });
			break;
		case 'no_record':
			chips.push({ key: 'domains.status.no_record', tone: 'danger' });
			break;
		default:
			chips.push({ key: 'domains.status.unknown', tone: 'muted' });
	}

	if (status.caa === 'caa_blocks') chips.push({ key: 'domains.status.caa_blocks', tone: 'danger' });

	if (https && status.certificate === 'certificate_ok')
		chips.push({ key: 'domains.status.certificate_ok', tone: 'success' });
	else if (https && status.certificate === 'default_certificate')
		chips.push({ key: 'domains.status.default_certificate', tone: 'warning' });
	else if (https && status.certificate === 'unreachable')
		chips.push({ key: 'domains.status.unreachable', tone: 'warning' });

	return chips;
}

/** Dot colour summing a domain up in the settings list: the worst of its chips. */
export function worstTone(chips: DomainChip[]): ChipTone {
	for (const tone of ['danger', 'warning', 'success'] as const) {
		if (chips.some((c) => c.tone === tone)) return tone;
	}
	return 'muted';
}

export type DnsRecord = { hostname: string; type: 'CNAME' | 'A / AAAA'; values: string[] };

/**
 * The record to create for a domain: a CNAME to the service's default address, or, for a bare
 * domain (which can't have a CNAME), A/AAAA records with the addresses that address resolves to.
 * A bare domain is guessed as two labels (example.com); three-label public suffixes such as
 * example.co.uk read as subdomains, which the person corrects at their provider if needed.
 */
export function recordFor(hostname: string, defaultHost: string, addresses: string[]): DnsRecord {
	if (hostname.split('.').length <= 2) {
		return { hostname, type: 'A / AAAA', values: addresses.length ? addresses : ['—'] };
	}
	return { hostname, type: 'CNAME', values: [defaultHost] };
}
