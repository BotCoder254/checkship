<!--
	Auth layout (DESIGN.md §5): two full-height columns from 1024px. The dithered art panel sits on
	`canvas`, the content on `surface`; the change of tone is the only boundary, there is no line.
	Below 1024px the panel is hidden and the content is centred alone.
-->
<script lang="ts">
	import DitherImage from '#lib/components/dither-image.svelte';
	import l from '#lib/localization/index.ts';

	let { children } = $props();
</script>

<div class="grid min-h-svh bg-surface font-sans text-ink lg:grid-cols-2">
	<div data-slot="auth-art" class="relative hidden bg-canvas lg:block">
		<DitherImage src="/auth/art.svg" class="absolute inset-0" />
		<p class="absolute bottom-12 left-12 max-w-84 text-lg/7 font-semibold tracking-tight text-ink">
			{l.translate('auth.tagline')}
		</p>
	</div>

	<main class="flex items-center justify-center px-6 py-12">
		{@render children()}
	</main>
</div>
