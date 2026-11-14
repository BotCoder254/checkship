/**
 * One runtime logs stream (GET /api/v1/apps/{id}/logs), read with fetch rather than EventSource:
 * EventSource hides the HTTP status, and the page needs it to tell "never deployed" (404),
 * "logs unavailable" (409) and "too many streams" (429) apart. It reconnects on its own and
 * resumes from the last line it received.
 */
import {
	createSSEParser,
	type LogLine,
	type ServiceEvent,
	type SSEMessage
} from './runtime-logs.ts';

export type ConnectionStatus = 'connecting' | 'live' | 'reconnecting' | 'stopped';

/** Why the stream stopped and won't reconnect by itself. */
export type ConnectionProblem =
	'no_containers' | 'logs_unavailable' | 'too_many_streams' | 'not_found' | 'failed';

type Handlers = {
	lines(batch: LogLine[]): void;
	event(event: ServiceEvent): void;
};

const RETRY_DELAYS_MS = [1000, 2000, 5000, 10000];

export class LogsConnection {
	status = $state<ConnectionStatus>('connecting');
	problem = $state<ConnectionProblem>();
	/** Time of the last line received, where a reconnection resumes. */
	resumeFrom = $state<string>();

	private controller?: AbortController;
	private stopped = false;
	private attempt = 0;

	constructor(
		private readonly url: string,
		private readonly handlers: Handlers
	) {}

	start() {
		this.stopped = false;
		this.problem = undefined;
		this.status = 'connecting';
		void this.run();
	}

	stop() {
		this.stopped = true;
		this.controller?.abort();
	}

	/** Tries again after a problem (too many streams, for example). */
	retry() {
		this.stop();
		this.attempt = 0;
		this.start();
	}

	private async run() {
		while (!this.stopped) {
			const ended = await this.connect();
			if (ended || this.stopped) return;

			this.status = 'reconnecting';
			const delay = RETRY_DELAYS_MS[Math.min(this.attempt++, RETRY_DELAYS_MS.length - 1)];
			await new Promise((resolve) => setTimeout(resolve, delay));
		}
	}

	/** One connection. Returns true when the stream must not be opened again. */
	private async connect(): Promise<boolean> {
		this.controller = new AbortController();
		// this.url already has a query string.
		let url = this.url;
		const headers: Record<string, string> = { Accept: 'text/event-stream' };
		if (this.resumeFrom) {
			url += `&since=${encodeURIComponent(this.resumeFrom)}`;
			headers['Last-Event-ID'] = this.resumeFrom;
		}

		let response: Response;
		try {
			response = await fetch(url, {
				headers,
				signal: this.controller.signal,
				cache: 'no-store'
			});
		} catch {
			return this.stopped;
		}

		if (!response.ok) return this.fail(response);

		this.status = 'live';
		this.attempt = 0;

		let problem: ConnectionProblem | undefined;
		const parser = createSSEParser((message) => {
			problem = this.handle(message) ?? problem;
		});

		try {
			const reader = response.body!.pipeThrough(new TextDecoderStream()).getReader();
			for (;;) {
				const { value, done } = await reader.read();
				if (done) break;
				parser.push(value);
				if (problem) {
					await reader.cancel();
					break;
				}
			}
		} catch {
			// Network drop or abort: reconnect unless stopped.
		}

		if (problem) {
			this.problem = problem;
			this.status = 'stopped';
			return true;
		}

		return this.stopped;
	}

	private handle(message: SSEMessage): ConnectionProblem | undefined {
		try {
			if (message.event === 'logs') {
				const batch = JSON.parse(message.data) as LogLine[];
				if (message.id) this.resumeFrom = message.id;
				this.handlers.lines(batch);
			} else if (message.event === 'state') {
				this.handlers.event(JSON.parse(message.data) as ServiceEvent);
			} else if (message.event === 'error') {
				const code = (JSON.parse(message.data) as { code?: string }).code;
				return code === 'logs_unavailable' ? 'logs_unavailable' : 'failed';
			}
		} catch {
			// A message that can't be read is skipped.
		}
		return undefined;
	}

	private async fail(response: Response): Promise<boolean> {
		let code: string | undefined;
		try {
			code = ((await response.json()) as { code?: string }).code;
		} catch {
			// No JSON body.
		}

		// Server errors (a target that can't be reached, for example) are worth a few more tries;
		// the others won't change by themselves.
		if (response.status >= 500 && this.attempt < RETRY_DELAYS_MS.length) return false;

		this.problem =
			code === 'no_containers' || code === 'logs_unavailable' || code === 'too_many_streams'
				? code
				: response.status === 404
					? 'not_found'
					: 'failed';
		this.status = 'stopped';
		return true;
	}
}
