<!--
	Language menu, next to the account avatar in the title bar. A ghost icon button with a tooltip;
	the menu lists every language the dashboard has, each in its own name, the current one ticked.
	Choosing one saves it for this browser and reloads the page in that language (the same as
	Profile › Interface).
-->
<script lang="ts">
	import LanguageCircleIcon from '@hugeicons/core-free-icons/LanguageCircleIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import Icon from '#lib/components/icon.svelte';
	import { iconButtonClass } from '#lib/components/icon-button.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import l, { type AppLocales, type LocaleCode } from '#lib/localization/index.ts';

	const current = l.locale();
	const languages = l.locales().map((locale) => ({
		code: locale.code as LocaleCode<AppLocales>,
		name: locale.displayName
	}));
	const currentName = languages.find((language) => language.code === current)?.name ?? current;
	const label = l.translate('language.menu', [currentName]);
</script>

<DropdownMenu.Root>
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props: tooltipProps })}
				<DropdownMenu.Trigger {...tooltipProps}>
					{#snippet child({ props })}
						<button
							{...props}
							type="button"
							aria-label={label}
							class={iconButtonClass({ tone: 'frame', touch: true })}
						>
							<Icon icon={LanguageCircleIcon} class="size-5" />
						</button>
					{/snippet}
				</DropdownMenu.Trigger>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content side="bottom">{label}</Tooltip.Content>
	</Tooltip.Root>
	<DropdownMenu.Content align="end" sideOffset={6} class="w-52 p-1.5">
		<DropdownMenu.Label class="px-2.5 pt-1.5 pb-1 text-xs font-semibold text-muted-foreground">
			{l.translate('language.title')}
		</DropdownMenu.Label>
		{#each languages as language (language.code)}
			{@const selected = language.code === current}
			<DropdownMenu.Item
				class="h-9 gap-3 px-2.5"
				aria-current={selected ? 'true' : undefined}
				onSelect={() => {
					if (!selected) l.locale(language.code);
				}}
			>
				<Icon icon={LanguageCircleIcon} class="size-4.5" />
				<span class="flex-1" lang={language.code}>{language.name}</span>
				<span class="font-mono text-xs text-muted-foreground uppercase">{language.code}</span>
				{#if selected}
					<Icon icon={Tick02Icon} class="size-4 text-primary" />
				{:else}
					<span class="size-4" aria-hidden="true"></span>
				{/if}
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>
