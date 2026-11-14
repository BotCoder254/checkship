/**
 * Copying text. `navigator.clipboard` only exists in a secure context (HTTPS or localhost),
 * and checkship is often reached over plain HTTP (`http://<ip>:8080`). There, a hidden textarea
 * with `document.execCommand('copy')` still works in every major browser (deprecated, kept for
 * this case). Call it from the click handler: browsers only allow copying right after a user action.
 */

export type ClipboardEnv = {
	isSecureContext: boolean;
	clipboard?: { writeText(text: string): Promise<void> };
	document?: Pick<Document, 'createElement' | 'body' | 'execCommand' | 'activeElement'>;
};

function browserEnv(): ClipboardEnv {
	return {
		isSecureContext: typeof window !== 'undefined' && window.isSecureContext,
		clipboard: typeof navigator !== 'undefined' ? navigator.clipboard : undefined,
		document: typeof document !== 'undefined' ? document : undefined
	};
}

function copyWithTextarea(text: string, doc: NonNullable<ClipboardEnv['document']>): boolean {
	const focused = doc.activeElement as HTMLElement | null;
	const area = doc.createElement('textarea');
	area.value = text;
	area.setAttribute('readonly', '');
	// Off screen but still selectable; a hidden element can't be selected.
	area.style.position = 'fixed';
	area.style.top = '0';
	area.style.left = '-9999px';
	area.style.opacity = '0';
	doc.body.appendChild(area);
	area.select();

	try {
		return doc.execCommand('copy');
	} catch {
		return false;
	} finally {
		area.remove();
		focused?.focus?.();
	}
}

/** Copies the text and says whether it worked. */
export async function copyText(text: string, env: ClipboardEnv = browserEnv()): Promise<boolean> {
	if (env.isSecureContext && env.clipboard) {
		try {
			await env.clipboard.writeText(text);
			return true;
		} catch {
			// Permission refused or no focus: try the older way below.
		}
	}

	return env.document ? copyWithTextarea(text, env.document) : false;
}
