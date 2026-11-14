<script lang="ts">
	// Theme, Tailwind and the legacy global CSS (reset + src/assets/styles.css, in a cascade layer).
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '#lib/components/ui/sonner/index.js';

	let { children } = $props();
</script>

<!-- Dark by default: the pre-migration pages are dark-only. Switch to "system" once they are migrated. -->
<ModeWatcher defaultMode="dark" />
<Toaster position="bottom-right" visibleToasts={3} />

{@render children()}
