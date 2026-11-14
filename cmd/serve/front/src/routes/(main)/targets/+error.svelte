<script lang="ts">
	import ErrorState from '#lib/components/error-state.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
</script>

<ErrorState
	what="target"
	back={{ href: routes.targets, label: l.translate('errors.back.targets') }}
/>
