<script lang="ts">
	// Theme and Tailwind (DESIGN.md §13.3).
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '#lib/components/ui/sonner/index.js';

	let { children } = $props();
</script>

<!-- Dark by default (owner). app.html seeds it, because mode-watcher 1.1.0 ignores defaultMode. -->
<ModeWatcher defaultMode="dark" />
<Toaster position="bottom-right" visibleToasts={3} />

{@render children()}
