<script lang="ts">
	import { mode } from 'mode-watcher';
	import { Toaster as Sonner, type ToasterProps as SonnerProps } from 'svelte-sonner';
	import Icon from '#lib/components/icon.svelte';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';

	let { ...restProps }: SonnerProps = $props();
</script>

<Sonner
	theme={mode.current}
	class="toaster group"
	style="--normal-bg: var(--color-popover); --normal-text: var(--color-popover-foreground); --normal-border: var(--color-border);"
	toastOptions={{
		classes: {
			toast: 'cn-toast'
		}
	}}
	{...restProps}
>
	{#snippet loadingIcon()}
		<Icon icon={Loading03Icon} class="size-4 animate-spin" />
	{/snippet}
	{#snippet successIcon()}
		<Icon icon={CheckmarkCircle02Icon} class="size-4" />
	{/snippet}
	{#snippet errorIcon()}
		<Icon icon={CancelCircleIcon} class="size-4" />
	{/snippet}
	{#snippet infoIcon()}
		<Icon icon={InformationCircleIcon} class="size-4" />
	{/snippet}
	{#snippet warningIcon()}
		<Icon icon={Alert02Icon} class="size-4" />
	{/snippet}
</Sonner>
