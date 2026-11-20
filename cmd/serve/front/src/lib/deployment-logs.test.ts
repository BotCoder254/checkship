import { describe, expect, it } from 'vitest';
import { applyLogMessage, compactProgress, initialLogStreamState } from './deployment-logs.ts';

describe('applyLogMessage', () => {
	it('appends chunks and remembers the offset', () => {
		let state = applyLogMessage(initialLogStreamState, {
			event: 'logs',
			id: '11',
			data: JSON.stringify({ text: '[STEP] one\n' })
		});
		state = applyLogMessage(state, {
			event: 'logs',
			id: '22',
			data: JSON.stringify({ text: '[STEP] two\n' })
		});

		expect(state).toEqual({ text: '[STEP] one\n[STEP] two\n', offset: 22, ended: false });
	});

	it('clears the log on reset and ends on end', () => {
		let state = { text: 'old', offset: 500, ended: false };
		state = applyLogMessage(state, { event: 'reset', id: '0', data: '{}' });
		expect(state).toEqual({ text: '', offset: 0, ended: false });

		state = applyLogMessage(state, { event: 'end', data: '{}' });
		expect(state.ended).toBe(true);
	});

	it('ignores unreadable and unknown messages', () => {
		const state = { text: 'a', offset: 1, ended: false };
		expect(applyLogMessage(state, { event: 'logs', id: '9', data: 'not json' })).toBe(state);
		expect(applyLogMessage(state, { event: 'ping', data: '{}' })).toBe(state);
	});

	it('keeps the offset when the id is missing or invalid', () => {
		const state = applyLogMessage(
			{ text: '', offset: 5, ended: false },
			{ event: 'logs', id: 'x', data: JSON.stringify({ text: 'b' }) }
		);
		expect(state.offset).toBe(5);
	});
});

describe('compactProgress', () => {
	it('keeps the latest status of each layer at its first place', () => {
		const log = [
			'[STEP] launching docker compose project (pulling, building and running)',
			' api Pulling',
			' 9d1f1a2f4c71 Pulling fs layer',
			' 3c2e8b7d0aa1 Pulling fs layer',
			' 9d1f1a2f4c71 Downloading [==>      ]  10MB/115MB',
			' 3c2e8b7d0aa1 Downloading [=>       ]  5MB/98MB',
			' 9d1f1a2f4c71 Downloading [=====>   ]  80MB/115MB',
			' 3c2e8b7d0aa1 Download complete',
			''
		].join('\n');

		const { text, collapsed } = compactProgress(log);

		expect(collapsed).toBe(4);
		expect(text).toBe(
			[
				'[STEP] launching docker compose project (pulling, building and running)',
				' api Pulling',
				' 9d1f1a2f4c71 Downloading [=====>   ]  80MB/115MB',
				' 3c2e8b7d0aa1 Download complete',
				''
			].join('\n')
		);
	});

	it('starts a new run after any other line', () => {
		const log = [
			' aaaaaaaaaaaa Downloading 1',
			'[WARN] something',
			' aaaaaaaaaaaa Downloading 2'
		].join('\n');

		expect(compactProgress(log)).toEqual({ text: log, collapsed: 0 });
	});

	it('collapses BuildKit layer downloads per step and layer', () => {
		const log = [
			'#4 [1/2] FROM docker.io/library/python:3.13-alpine',
			'#4 sha256:aaa 0B / 12.67MB 0.2s',
			'#4 sha256:bbb 412.00kB / 1.2MB 3.5s',
			'#4 sha256:aaa 6.30MB / 12.67MB 5.3s',
			'#4 sha256:aaa 12.67MB / 12.67MB 44.2s done',
			'#5 0.311 hello from the build'
		].join('\n');

		expect(compactProgress(log).text).toBe(
			[
				'#4 [1/2] FROM docker.io/library/python:3.13-alpine',
				'#4 sha256:aaa 12.67MB / 12.67MB 44.2s done',
				'#4 sha256:bbb 412.00kB / 1.2MB 3.5s',
				'#5 0.311 hello from the build'
			].join('\n')
		);
	});

	it('leaves logs without layers untouched', () => {
		const log = '[STEP] one\n[INFO] two\n';
		expect(compactProgress(log)).toEqual({ text: log, collapsed: 0 });
	});
});
