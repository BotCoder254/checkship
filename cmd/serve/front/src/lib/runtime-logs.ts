/**
 * Runtime logs (DESIGN.md §11.7 Logs tab): the types sent by GET /api/v1/apps/{id}/logs and
 * /services, a parser for its server-sent events, the line buffer behind the console and the
 * formats used to copy and download lines. No DOM here, so all of it is unit tested.
 */

export type LogStream = 'stdout' | 'stderr';

export type LogLine = {
	time: string;
	service: string;
	container: string;
	stream: LogStream;
	text: string;
	truncated?: boolean;
};

export type ServiceAction = 'start' | 'die' | 'oom' | 'health' | 'destroy';

export type ServiceEvent = {
	time: string;
	service: string;
	container: string;
	action: ServiceAction;
	exit_code: number | null;
	health: string | null;
};

export type ServiceState = {
	service: string;
	container: string;
	image: string;
	state: string;
	status: string;
	health: string | null;
	restart_count: number;
	exit_code: number | null;
	started_at: string | null;
};

/** A line as the console shows it. */
export type ViewLine = LogLine & { id: number; error: boolean };

/** The console keeps the newest lines only. */
export const MAX_LINES = 5000;
export const TAIL_CHOICES = [100, 200, 1000, 5000] as const;
export const DEFAULT_TAIL = 200;

// CSI and OSC sequences, as matched by the ansi-regex package.
const ANSI =
	// eslint-disable-next-line no-control-regex
	/[\u001B\u009B][[\]()#;?]*(?:(?:(?:(?:;[-a-zA-Z\d/#&.:=?%@~_]+)*|[a-zA-Z\d]+(?:;[-a-zA-Z\d/#&.:=?%@~_]*)*)?\u0007)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PR-TZcf-nq-uy=><~]))/g;
// eslint-disable-next-line no-control-regex
const CONTROL = /[\u0000-\u0008\u000B-\u001F\u007F]/g;

/** Removes colour and cursor codes: the palette never changes (DESIGN.md §3). */
export const stripAnsi = (text: string) => text.replace(ANSI, '').replace(CONTROL, '');

/** Same rule as the deployment logs (`highlightLog`): these lines are shown in danger. */
export const ERROR_PATTERN = /\b(error|failed|fatal|panic|exception)\b/i;

// ---------------------------------------------------------------------------------------------
// Server-sent events
// ---------------------------------------------------------------------------------------------

export type SSEMessage = { event: string; data: string; id?: string };

/**
 * Parses a text/event-stream body fed in chunks of any size. Comments (`: ping`) and `retry`
 * fields are ignored.
 */
export function createSSEParser(onMessage: (message: SSEMessage) => void) {
	let buffer = '';
	let event = '';
	let data: string[] = [];
	let id: string | undefined;

	function dispatch() {
		if (data.length > 0) onMessage({ event: event || 'message', data: data.join('\n'), id });
		event = '';
		data = [];
		id = undefined;
	}

	function line(text: string) {
		if (text === '') return dispatch();
		if (text.startsWith(':')) return;

		const colon = text.indexOf(':');
		const field = colon < 0 ? text : text.slice(0, colon);
		let value = colon < 0 ? '' : text.slice(colon + 1);
		if (value.startsWith(' ')) value = value.slice(1);

		if (field === 'event') event = value;
		else if (field === 'data') data.push(value);
		else if (field === 'id') id = value;
	}

	return {
		push(chunk: string) {
			buffer += chunk;
			let index: number;
			while ((index = buffer.search(/\r\n|\r|\n/)) >= 0) {
				const end = buffer.startsWith('\r\n', index) ? 2 : 1;
				line(buffer.slice(0, index));
				buffer = buffer.slice(index + end);
			}
		}
	};
}

// ---------------------------------------------------------------------------------------------
// Line buffer
// ---------------------------------------------------------------------------------------------

const lineKey = (l: LogLine) => `${l.time}\u0000${l.container}\u0000${l.stream}\u0000${l.text}`;

/**
 * Keeps the newest MAX_LINES lines. After a reconnection the server reads back from the last
 * line's time, so lines already shown can come again: they are dropped.
 */
export class LineBuffer {
	lines: ViewLine[] = [];
	/** True once older lines have been dropped. */
	dropped = false;
	private nextId = 0;

	constructor(private readonly max = MAX_LINES) {}

	/** Adds a batch and returns how many lines were really added. */
	append(batch: LogLine[]): number {
		if (batch.length === 0) return 0;

		const newest = this.lines.at(-1)?.time;
		let seen: Set<string> | undefined;

		// Only lines not newer than what is shown can be repeats.
		if (newest !== undefined && batch[0].time <= newest) {
			const from = batch[0].time;
			seen = new Set();
			for (let i = this.lines.length - 1; i >= 0 && this.lines[i].time >= from; i--) {
				seen.add(lineKey(this.lines[i]));
			}
		}

		const added: ViewLine[] = [];

		for (const l of batch) {
			if (seen?.has(lineKey(l))) continue;
			const text = stripAnsi(l.text);
			added.push({ ...l, text, id: this.nextId++, error: ERROR_PATTERN.test(text) });
		}

		if (added.length === 0) return 0;

		let next = this.lines.concat(added);
		if (next.length > this.max) {
			next = next.slice(next.length - this.max);
			this.dropped = true;
		}
		this.lines = next;

		return added.length;
	}

	clear() {
		this.lines = [];
		this.dropped = false;
	}
}

// ---------------------------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------------------------

export type Search = { include: string[]; exclude: string[] };

/** Words must all appear; a word starting with `-` must not. Case doesn't matter. */
export function parseSearch(query: string): Search {
	const search: Search = { include: [], exclude: [] };

	for (const word of query.toLowerCase().split(/\s+/)) {
		if (!word || word === '-') continue;
		if (word.startsWith('-')) search.exclude.push(word.slice(1));
		else search.include.push(word);
	}

	return search;
}

export type Filters = { service?: string; search: Search; errorsOnly: boolean };

export function matches(line: ViewLine, filters: Filters): boolean {
	if (filters.service && line.service !== filters.service) return false;
	if (filters.errorsOnly && !line.error && line.stream !== 'stderr') return false;

	const { include, exclude } = filters.search;
	if (include.length === 0 && exclude.length === 0) return true;

	const text = line.text.toLowerCase();
	return include.every((w) => text.includes(w)) && !exclude.some((w) => text.includes(w));
}

/** Splits a text around the searched words so they can be highlighted. */
export function highlightParts(text: string, words: string[]): { text: string; match: boolean }[] {
	const terms = words.filter(Boolean);
	if (terms.length === 0) return [{ text, match: false }];

	const escaped = terms.map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
	const pattern = new RegExp(`(${escaped.join('|')})`, 'gi');

	return text
		.split(pattern)
		.filter((part) => part !== '')
		.map((part) => ({ text: part, match: terms.some((w) => w === part.toLowerCase()) }));
}

// ---------------------------------------------------------------------------------------------
// Formats
// ---------------------------------------------------------------------------------------------

const pad = (n: number, size = 2) => String(n).padStart(size, '0');

/** Local time with milliseconds: 23:41:02.118 */
export function displayTime(iso: string): string {
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return '';
	return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`;
}

export type LineFormat = { times: boolean; service: boolean };

/** One line as copied: what the console shows, time and service included when shown. */
export function formatLine(line: LogLine, format: LineFormat): string {
	const parts: string[] = [];
	if (format.times) parts.push(displayTime(line.time));
	if (format.service) parts.push(line.service);
	parts.push(line.text);
	return parts.join(' ');
}

export const copyAll = (lines: LogLine[], format: LineFormat) =>
	lines.map((l) => formatLine(l, format)).join('\n');

/** The lines joined into a single line, to paste in a chat or an issue. */
export const copyOneLine = (lines: LogLine[], format: LineFormat) =>
	lines
		.map((l) => formatLine(l, format).replace(/\s+/g, ' ').trim())
		.filter(Boolean)
		.join(' | ');

/** A downloaded file keeps full times and services, whatever the view shows. */
export const toLogFile = (lines: LogLine[]) =>
	lines.map((l) => `${l.time} ${l.service} ${l.stream} ${l.text}`).join('\n') + '\n';

export function logFileName(app: string, environment: string, at = new Date()): string {
	const stamp = `${at.getFullYear()}${pad(at.getMonth() + 1)}${pad(at.getDate())}-${pad(at.getHours())}${pad(at.getMinutes())}`;
	return `${app}-${environment}-${stamp}.log`;
}
