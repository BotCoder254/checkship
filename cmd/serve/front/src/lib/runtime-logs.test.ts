import { describe, expect, it } from 'vitest';
import {
	LineBuffer,
	copyAll,
	copyOneLine,
	createSSEParser,
	highlightParts,
	logFileName,
	matches,
	parseSearch,
	stripAnsi,
	toLogFile,
	type LogLine,
	type SSEMessage
} from './runtime-logs.ts';

const line = (time: string, text: string, extra: Partial<LogLine> = {}): LogLine => ({
	time,
	service: 'web',
	container: 'web-1',
	stream: 'stdout',
	text,
	...extra
});

describe('createSSEParser', () => {
	it('reads events split across chunks and skips comments', () => {
		const messages: SSEMessage[] = [];
		const parser = createSSEParser((m) => messages.push(m));

		parser.push('retry: 5000\n\nid: 2026-11-14T23:41:03Z\nevent: lo');
		parser.push('gs\ndata: [1]\n\n: ping\n\nevent: state\r\ndata: {"a":1}\r\n\r\n');

		expect(messages).toEqual([
			{ event: 'logs', data: '[1]', id: '2026-11-14T23:41:03Z' },
			{ event: 'state', data: '{"a":1}', id: undefined }
		]);
	});
});

describe('LineBuffer', () => {
	it('strips colour codes and marks error lines', () => {
		const buffer = new LineBuffer();

		buffer.append([line('a', '\u001b[32mready\u001b[0m'), line('b', 'Error: connect ETIMEDOUT')]);

		expect(buffer.lines.map((l) => [l.text, l.error])).toEqual([
			['ready', false],
			['Error: connect ETIMEDOUT', true]
		]);
	});

	it('drops lines read again after a reconnection', () => {
		const buffer = new LineBuffer();
		buffer.append([line('2026-01-01T00:00:01Z', 'one'), line('2026-01-01T00:00:02Z', 'two')]);

		const added = buffer.append([
			line('2026-01-01T00:00:02Z', 'two'),
			line('2026-01-01T00:00:02Z', 'two again'),
			line('2026-01-01T00:00:03Z', 'three')
		]);

		expect(added).toBe(2);
		expect(buffer.lines.map((l) => l.text)).toEqual(['one', 'two', 'two again', 'three']);
	});

	it('keeps the newest lines and says older ones were dropped', () => {
		const buffer = new LineBuffer(3);

		buffer.append(['1', '2', '3', '4', '5'].map((n) => line(`2026-01-01T00:00:0${n}Z`, n)));

		expect(buffer.lines.map((l) => l.text)).toEqual(['3', '4', '5']);
		expect(buffer.dropped).toBe(true);
	});
});

describe('filters', () => {
	const lines = new LineBuffer();
	lines.append([
		line('1', 'POST /api/checkout 502 upstream timeout'),
		line('2', 'GET /healthz 200 timeout=5s'),
		line('3', 'worker ready', { service: 'worker', stream: 'stderr' }),
		line('4', 'GET / 200')
	]);

	const shown = (query: string, errorsOnly = false, service?: string) =>
		lines.lines
			.filter((l) => matches(l, { search: parseSearch(query), errorsOnly, service }))
			.map((l) => l.time);

	it('keeps lines with every word and hides those with a -word', () => {
		expect(shown('timeout')).toEqual(['1', '2']);
		expect(shown('timeout -healthz')).toEqual(['1']);
		expect(shown('TIMEOUT -')).toEqual(['1', '2']);
	});

	it('keeps errors and stderr lines with errors only, and filters on the service', () => {
		lines.append([line('5', 'FATAL: too many connections')]);
		expect(shown('', true)).toEqual(['3', '5']);
		expect(shown('', false, 'worker')).toEqual(['3']);
	});

	it('splits a text around the searched words', () => {
		expect(highlightParts('upstream Timeout here', ['timeout'])).toEqual([
			{ text: 'upstream ', match: false },
			{ text: 'Timeout', match: true },
			{ text: ' here', match: false }
		]);
		expect(highlightParts('a+b', ['+'])).toEqual([
			{ text: 'a', match: false },
			{ text: '+', match: true },
			{ text: 'b', match: false }
		]);
	});
});

describe('formats', () => {
	const lines = [
		line('2026-11-14T23:41:02.118Z', 'POST /api/checkout 502  '),
		line('2026-11-14T23:41:02.320Z', '  Error: connect\tETIMEDOUT', { stream: 'stderr' })
	];

	it('copies lines one per line or joined into one line', () => {
		const format = { times: false, service: true };

		expect(copyAll(lines, format)).toBe(
			'web POST /api/checkout 502  \nweb   Error: connect\tETIMEDOUT'
		);
		expect(copyOneLine(lines, format)).toBe(
			'web POST /api/checkout 502 | web Error: connect ETIMEDOUT'
		);
	});

	it('writes full times, services and streams to a downloaded file', () => {
		expect(toLogFile(lines)).toBe(
			'2026-11-14T23:41:02.118Z web stdout POST /api/checkout 502  \n2026-11-14T23:41:02.320Z web stderr   Error: connect\tETIMEDOUT\n'
		);
		expect(logFileName('welcome', 'production', new Date(2026, 10, 14, 23, 41))).toBe(
			'welcome-production-20261114-2341.log'
		);
	});

	it('removes colour codes and other control characters', () => {
		expect(
			stripAnsi('\u001b]8;;http://x\u0007link\u001b]8;;\u0007 \u001b[1;31mred\u001b[0m\u0007')
		).toBe('link red');
	});
});
