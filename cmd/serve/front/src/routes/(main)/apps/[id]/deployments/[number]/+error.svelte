<script lang="ts">
	import { page } from '$app/state';
	import ErrorState from '#lib/components/error-state.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
</script>

<ErrorState
	what="deployment"
	back={{ href: routes.app(page.params.id ?? ''), label: l.translate('errors.back.app') }}
/>
