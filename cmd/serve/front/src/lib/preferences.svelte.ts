/**
 * Per-browser interface preferences (DESIGN.md §5): sidebar collapsed, content width and the
 * recently opened applications. Conveniences only, so storage that is blocked or empty (private
 * mode) simply falls back to the defaults.
 */

const SIDEBAR_KEY = 'checkship:sidebar-collapsed';
const WIDTH_KEY = 'checkship:content-width';
const RECENT_KEY = 'checkship:recent-apps';
const RECENT_MAX = 5;

export const CONTENT_WIDTHS = {
	compact: '840px',
	default: '1020px',
	wide: '1160px'
} as const;

export type ContentWidth = keyof typeof CONTENT_WIDTHS;

function read(key: string): string | null {
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}

function write(key: string, value: string) {
	try {
		localStorage.setItem(key, value);
	} catch {
		// Storage blocked: the preference lasts for this page only.
	}
}

function isContentWidth(value: string | null): value is ContentWidth {
	return value !== null && value in CONTENT_WIDTHS;
}

function readRecent(): string[] {
	try {
		const value = JSON.parse(read(RECENT_KEY) ?? '[]');
		return Array.isArray(value)
			? value.filter((id) => typeof id === 'string').slice(0, RECENT_MAX)
			: [];
	} catch {
		return [];
	}
}

class Preferences {
	sidebarCollapsed = $state(read(SIDEBAR_KEY) === 'true');
	contentWidth = $state<ContentWidth>(
		isContentWidth(read(WIDTH_KEY)) ? (read(WIDTH_KEY) as ContentWidth) : 'default'
	);
	recentApps = $state<string[]>(readRecent());

	toggleSidebar() {
		this.sidebarCollapsed = !this.sidebarCollapsed;
		write(SIDEBAR_KEY, String(this.sidebarCollapsed));
	}

	setContentWidth(value: ContentWidth) {
		this.contentWidth = value;
		write(WIDTH_KEY, value);
	}

	/** Moves an application to the top of the Recent list. */
	rememberApp(id: string) {
		this.recentApps = [id, ...this.recentApps.filter((other) => other !== id)].slice(0, RECENT_MAX);
		write(RECENT_KEY, JSON.stringify(this.recentApps));
	}
}

export const preferences = new Preferences();
