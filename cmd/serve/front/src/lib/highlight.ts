/**
 * Syntax highlighting (DESIGN.md §11.6) with Shiki: the core, the JavaScript regex engine (no
 * WASM) and only the grammars the dashboard shows. The theme is Shiki's css-variables theme, so
 * every colour is a palette token set in layout.css and follows light and dark.
 *
 * Everything loads on first use, so pages without code never download Shiki.
 */
import type { HighlighterCore } from 'shiki/core';

export type CodeLang = 'yaml' | 'json' | 'shellscript' | 'dotenv' | 'log' | 'text';

let highlighter: Promise<HighlighterCore> | undefined;

function load(): Promise<HighlighterCore> {
	highlighter ??= (async () => {
		const [{ createHighlighterCore, createCssVariablesTheme }, { createJavaScriptRegexEngine }] =
			await Promise.all([import('shiki/core'), import('shiki/engine/javascript')]);

		return createHighlighterCore({
			themes: [createCssVariablesTheme({ name: 'checkship', variablePrefix: '--shiki-' })],
			langs: [
				import('shiki/langs/yaml.mjs'),
				import('shiki/langs/json.mjs'),
				import('shiki/langs/shellscript.mjs'),
				import('shiki/langs/dotenv.mjs')
			],
			engine: createJavaScriptRegexEngine()
		});
	})();

	return highlighter;
}

export const escapeHtml = (text: string) =>
	text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

/**
 * Logs have no grammar: the leading timestamp is faint and lines reporting an error are in danger.
 */
export function highlightLog(code: string): string {
	return code
		.split('\n')
		.map((line) => {
			const match = line.match(/^(\[?[\d:.TZ+-]{8,}\]?)(\s.*)?$/);
			const rest = match ? (match[2] ?? '') : line;
			const tone = /\b(error|failed|fatal|panic)\b/i.test(rest) ? 'log-error' : 'log-text';
			return (
				(match ? `<span class="log-time">${escapeHtml(match[1])}</span>` : '') +
				`<span class="${tone}">${escapeHtml(rest)}</span>`
			);
		})
		.join('\n');
}

/**
 * Returns the code as highlighted HTML lines (one `<span class="line">` per line, no wrapper), or
 * escaped text when the language has no grammar.
 */
export async function highlight(code: string, lang: CodeLang): Promise<string> {
	if (lang === 'log') return highlightLog(code);
	if (lang === 'text') return escapeHtml(code);

	const shiki = await load();
	const html = shiki.codeToHtml(code, { lang, theme: 'checkship' });

	// Keep the lines only: the caller draws the block (background, padding, font).
	return html.replace(/^<pre[^>]*><code>/, '').replace(/<\/code><\/pre>$/, '');
}
