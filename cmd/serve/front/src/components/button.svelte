<script lang="ts">
	import ButtonBase from '$components/button-base.svelte';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import type { ComponentProps } from 'svelte';

	export let text: AppTranslationsString;

	type $$Props = ComponentProps<ButtonBase> & {
		text: AppTranslationsString;
	};
</script>

<ButtonBase on:click {...$$restProps}>
	{l.translate(text)}
</ButtonBase>
