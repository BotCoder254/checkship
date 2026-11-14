<!-- Errors outside the shell (no session yet, or the shell itself failed): centred on the canvas. -->
<script lang="ts">
	import ErrorState from '#lib/components/error-state.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
</script>

<main class="mx-auto flex min-h-dvh w-full max-w-160 flex-col justify-center px-4">
	<ErrorState what="page" back={{ href: routes.apps, label: l.translate('errors.back.apps') }} />
</main>
