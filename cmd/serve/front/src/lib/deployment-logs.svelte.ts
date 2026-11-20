/**
 * The live log of one deployment (GET …/deployments/{number}/logs/stream), read with fetch
 * like the runtime logs (#lib/runtime-logs.svelte.ts) so the HTTP status is visible. It resumes
 * from the byte offset it reached after a drop, and gives up for the plain polled log after
 * four failures in a row (a proxy that buffers or blocks the stream, for example).
 */
import { applyLogMessage, initialLogStreamState, type LogStreamState } from './deployment-logs.ts';
import { createSSEParser } from './runtime-logs.ts';

export type DeploymentLogStatus = 'connecting' | 'live' | 'reconnecting' | 'ended' | 'fallback';

const RETRY_DELAYS_MS = [1000, 2000, 5000, 10000];

export class DeploymentLogStream {
	status = $state<DeploymentLogStatus>('connecting');
	text = $state('');

	private log: LogStreamState = initialLogStreamState;
	private controller?: AbortController;
	private stopped = false;
	private failures = 0;

	constructor(private readonly url: string) {}

	start() {
		this.stopped = false;
		this.status = 'connecting';
		void this.run();
	}

	stop() {
		this.stopped = true;
		this.controller?.abort();
	}

	private async run() {
		while (!this.stopped) {
			const ended = await this.connect();
			if (ended || this.stopped) return;

			if (this.failures >= RETRY_DELAYS_MS.length) {
				this.status = 'fallback';
				return;
			}

			this.status = 'reconnecting';
			const delay = RETRY_DELAYS_MS[Math.min(this.failures - 1, RETRY_DELAYS_MS.length - 1)];
			await new Promise((resolve) => setTimeout(resolve, Math.max(delay, 0)));
		}
	}

	/** One connection. Returns true when the stream must not be opened again. */
	private async connect(): Promise<boolean> {
		this.controller = new AbortController();
		const headers: Record<string, string> = { Accept: 'text/event-stream' };
		if (this.log.offset > 0) headers['Last-Event-ID'] = String(this.log.offset);

		let response: Response;
		try {
			response = await fetch(`${this.url}?offset=${this.log.offset}`, {
				headers,
				signal: this.controller.signal,
				cache: 'no-store'
			});
		} catch {
			this.failures++;
			return this.stopped;
		}

		// Gone, refused or over the limit: the polled log will say what there is to say.
		if (!response.ok || !response.body) {
			this.failures = RETRY_DELAYS_MS.length;
			return false;
		}

		this.status = 'live';

		const parser = createSSEParser((message) => {
			const before = this.log;
			this.log = applyLogMessage(this.log, message);
			if (this.log.text !== before.text) this.text = this.log.text;
			if (message.event === 'logs') this.failures = 0;
		});

		try {
			const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
			for (;;) {
				const { value, done } = await reader.read();
				if (done) break;
				parser.push(value);
				if (this.log.ended) {
					await reader.cancel();
					break;
				}
			}
		} catch {
			// Network drop or abort: reconnect unless stopped.
		}

		if (this.log.ended) {
			this.status = 'ended';
			return true;
		}

		this.failures++;
		return this.stopped;
	}
}
