<!--
	Environment variables of one environment (DESIGN.md §11.6), in a drawer opened from the
	settings page. One block per service: its name, then KEY=value lines highlighted as dotenv.
	A .env file dropped on the drop zone becomes a new block. Works on a copy: Cancel discards it,
	the save button hands the list back (and saves it right away on an existing application).
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import FileUploadIcon from '@hugeicons/core-free-icons/FileUploadIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { tick } from 'svelte';
	import CodeEditor from '#lib/components/code-editor.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import DropZone from '#lib/components/drop-zone.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import l from '#lib/localization/index.ts';
	import type { ServiceVariables } from '#lib/resources/apps.ts';

	type Props = {
		open: boolean;
		title: string;
		services: ServiceVariables[];
		/** Names of the services in the latest deployment, offered as help */
		known?: string[];
		saveLabel: string;
		onsave: (services: ServiceVariables[]) => Promise<unknown> | void;
	};

	let { open = $bindable(false), title, services, known = [], saveLabel, onsave }: Props = $props();

	let draft = $state<ServiceVariables[]>([]);
	let invalid = $state<number[]>([]);
	let saving = $state(false);
	let error = $state<string>();

	// A fresh copy each time the drawer opens.
	$effect(() => {
		if (open) {
			draft = services.map((s) => ({ ...s }));
			invalid = [];
			error = undefined;
		}
	});

	async function add(values = '', name = '') {
		draft.push({ name, values });
		await tick();
		document.getElementById(`vars-name-${draft.length - 1}`)?.focus();
	}

	async function fromFile(file: File) {
		const text = await file.text();
		const guess = file.name.replace(/^\.env\.?/, '').replace(/\.env$/, '');
		await add(text.trimEnd(), guess);
	}

	async function save() {
		const kept = draft.filter((s) => s.name.trim() || s.values.trim());
		invalid = kept.flatMap((s, i) => (s.name.trim() ? [] : [i]));
		if (invalid.length) {
			draft = kept;
			return;
		}
		saving = true;
		error = undefined;
		try {
			await onsave(kept.map((s) => ({ name: s.name.trim(), values: s.values })));
			open = false;
		} catch (ex) {
			error = ex instanceof Error ? ex.message : l.translate('unexpected_error');
		} finally {
			saving = false;
		}
	}
</script>

<Drawer bind:open {title} description={l.translate('vars.description')}>
	{#if error}
		<Alert.Root variant="destructive">
			<Icon icon={Alert02Icon} />
			<Alert.Title class="font-semibold">{error}</Alert.Title>
		</Alert.Root>
	{/if}

	<DropZone
		compact
		icon={FileUploadIcon}
		what={l.translate('vars.env.what')}
		hint={l.translate('vars.env.hint')}
		accept=".env,text/plain"
		onfile={fromFile}
	/>

	{#if draft.length === 0}
		<p class="text-sm text-muted-foreground">{l.translate('vars.empty')}</p>
	{/if}

	{#each draft as service, index (index)}
		<div class="flex flex-col gap-2.5">
			<div class="grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2.5">
				<div class="flex flex-col gap-1.5">
					<Label for="vars-name-{index}">{l.translate('vars.service')}</Label>
					<Input
						id="vars-name-{index}"
						bind:value={service.name}
						placeholder={l.translate('vars.service.placeholder')}
						aria-invalid={invalid.includes(index) || undefined}
						aria-describedby="vars-name-{index}-message"
					/>
				</div>
				<Button
					variant="ghost"
					size="icon"
					aria-label={l.translate('vars.remove', [service.name || l.translate('vars.service')])}
					onclick={() => draft.splice(index, 1)}
				>
					<Icon icon={Delete02Icon} />
				</Button>
			</div>
			<FieldMessage
				id="vars-name-{index}-message"
				error={invalid.includes(index) ? 'required' : undefined}
			>
				{#snippet help()}
					{#if known.length}{l.translate('vars.known', [known.join(', ')])}{/if}
				{/snippet}
			</FieldMessage>
			<Label for="vars-values-{index}">{l.translate('vars.values')}</Label>
			<CodeEditor
				id="vars-values-{index}"
				lang="dotenv"
				tone="canvas"
				rows={3}
				bind:value={service.values}
				placeholder="KEY=value"
			/>
		</div>
	{/each}

	<Button variant="outline" class="self-start" onclick={() => add()}>
		<Icon icon={Add01Icon} />
		{l.translate('vars.add')}
	</Button>

	{#snippet footer()}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		<Button loading={saving} onclick={save}>
			<Icon icon={Tick02Icon} />
			{saveLabel}
		</Button>
	{/snippet}
</Drawer>
