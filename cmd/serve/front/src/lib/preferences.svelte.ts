/**
 * Per-browser interface preferences (DESIGN.md §5): sidebar collapsed and the recently opened
 * applications. Conveniences only, so storage that is blocked or empty (private
 * mode) simply falls back to the defaults.
 */

const SIDEBAR_KEY = 'checkship:sidebar-collapsed';
const RECENT_KEY = 'checkship:recent-apps';
const RECENT_MAX = 5;

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
	recentApps = $state<string[]>(readRecent());

	toggleSidebar() {
		this.sidebarCollapsed = !this.sidebarCollapsed;
		write(SIDEBAR_KEY, String(this.sidebarCollapsed));
	}

	/** Moves an application to the top of the Recent list. */
	rememberApp(id: string) {
		this.recentApps = [id, ...this.recentApps.filter((other) => other !== id)].slice(0, RECENT_MAX);
		write(RECENT_KEY, JSON.stringify(this.recentApps));
	}
}

export const preferences = new Preferences();
