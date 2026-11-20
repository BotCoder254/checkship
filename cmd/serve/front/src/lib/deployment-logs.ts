/**
 * Pure helpers for the live deployment log (DESIGN.md §11.7 Deployment page): what the stream's
 * messages do to the log shown, and how image downloads collapse to one line per layer.
 */
import type { SSEMessage } from './runtime-logs.ts';

// ---------------------------------------------------------------------------------------------
// Stream messages (GET /api/v1/apps/{id}/deployments/{number}/logs/stream)
// ---------------------------------------------------------------------------------------------

export type LogStreamState = {
	/** The raw log received so far. */
	text: string;
	/** Byte offset reached in the log file, where a reconnection resumes. */
	offset: number;
	/** The deployment has finished and everything was received. */
	ended: boolean;
};

export const initialLogStreamState: LogStreamState = { text: '', offset: 0, ended: false };

/** Applies one server-sent event; unknown or unreadable ones change nothing. */
export function applyLogMessage(state: LogStreamState, message: SSEMessage): LogStreamState {
	switch (message.event) {
		case 'logs': {
			let chunk: string;
			try {
				chunk = (JSON.parse(message.data) as { text?: string }).text ?? '';
			} catch {
				return state;
			}
			const offset = Number(message.id);
			return {
				...state,
				text: state.text + chunk,
				offset: Number.isFinite(offset) && offset >= 0 ? offset : state.offset
			};
		}
		case 'reset':
			// The log on the server is not the one we had (shorter than where we were).
			return { ...state, text: '', offset: 0 };
		case 'end':
			return { ...state, ended: true };
		default:
			return state;
	}
}

// ---------------------------------------------------------------------------------------------
// Progress lines
// ---------------------------------------------------------------------------------------------

// Compose, one line per layer event: " 9d1f1a2f4c71 Downloading [====>   ]  83.4MB/115.5MB"
const composeLayer = /^\s?([0-9a-f]{12}) \S/;
// BuildKit base image layers: "#4 sha256:f9ab… 4.19MB / 12.67MB 16.7s" (and "… done")
const buildkitLayer = /^#(\d+) (sha256:[0-9a-f]+) [\d.]+\s?[kKMGT]?i?B \/ /;

function layerKey(line: string): string | undefined {
	const compose = composeLayer.exec(line);
	if (compose) return compose[1];
	const buildkit = buildkitLayer.exec(line);
	if (buildkit) return `#${buildkit[1]} ${buildkit[2]}`;
	return undefined;
}

/**
 * Shows each image layer once inside a run of layer lines, with its latest status, the way the
 * Docker CLI does in a terminal: a download reads as lines that move instead of hundreds of
 * them. Any other line ends the run, so steps, warnings and build output are never touched.
 * Only for display; copying uses the raw log. `collapsed` is the number of lines hidden.
 */
export function compactProgress(text: string): { text: string; collapsed: number } {
	const lines = text.split('\n');
	const out: string[] = [];
	let run = new Map<string, number>();
	let collapsed = 0;

	for (const line of lines) {
		const key = layerKey(line);

		if (key === undefined) {
			run = new Map();
			out.push(line);
			continue;
		}

		const at = run.get(key);
		if (at === undefined) {
			run.set(key, out.length);
			out.push(line);
		} else {
			out[at] = line;
			collapsed++;
		}
	}

	return { text: collapsed === 0 ? text : out.join('\n'), collapsed };
}
