/**
 * Live hints (DESIGN.md §11.10): one EventSource on GET /api/v1/events, shared by every component
 * that needs it. The server only says *what* changed ({ topic, id }); components refetch with the
 * user's session. When the stream can't connect (signed out, too many tabs, a proxy that buffers),
 * `status` says so and pages keep refreshing on their own timer.
 */
import { readonly, writable } from 'svelte/store';

export type LiveTopic = 'deployments' | 'targets' | 'apps' | 'backups';
export type LiveHint = { topic: LiveTopic; id?: string };
export type LiveStatus = 'connecting' | 'live' | 'offline';

/** Hints arriving within this window are reported once. */
const DEBOUNCE_MS = 300;

const status = writable<LiveStatus>('connecting');
const changes = writable<{ seq: number; topics: LiveTopic[] }>({ seq: 0, topics: [] });

let source: EventSource | undefined;
let users = 0;
let pending = new Set<LiveTopic>();
let timer: ReturnType<typeof setTimeout> | undefined;

function flush() {
	timer = undefined;
	const topics = [...pending];
	pending = new Set();
	changes.update((c) => ({ seq: c.seq + 1, topics }));
}

function open() {
	if (typeof EventSource === 'undefined') {
		status.set('offline');
		return;
	}

	status.set('connecting');
	source = new EventSource('/api/v1/events');
	source.onopen = () => status.set('live');
	// The browser retries on its own while the state is CONNECTING; CLOSED means it gave up
	// (HTTP error such as 401 or 429), so the page relies on its polling fallback.
	source.onerror = () =>
		status.set(source?.readyState === EventSource.CLOSED ? 'offline' : 'connecting');
	source.addEventListener('hint', (event) => {
		try {
			const hint = JSON.parse((event as MessageEvent<string>).data) as LiveHint;
			if (!['deployments', 'targets', 'apps', 'backups'].includes(hint.topic)) return;
			pending.add(hint.topic);
			timer ??= setTimeout(flush, DEBOUNCE_MS);
		} catch {
			// Ignore anything that isn't a hint.
		}
	});
}

function close() {
	source?.close();
	source = undefined;
	if (timer) clearTimeout(timer);
	timer = undefined;
	pending = new Set();
}

/** Start listening; the returned function stops (the stream closes with its last user). */
export function connectLive(): () => void {
	users++;
	if (users === 1) open();

	let stopped = false;
	return () => {
		if (stopped) return;
		stopped = true;
		users--;
		if (users === 0) close();
	};
}

export const liveStatus = readonly(status);
/** Increments (debounced) whenever the server reports a change; `topics` lists what changed. */
export const liveChanges = readonly(changes);
