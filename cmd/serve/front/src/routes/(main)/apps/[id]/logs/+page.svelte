<!--
	Logs tab (DESIGN.md §11.7): what the application's containers write, live, from one stream per
	page (GET /api/v1/apps/{id}/logs, every service merged in time order). The toolbar filters in the
	browser, except the environment and the number of lines to load, which reopen the stream. The
	strip above the console shows each container's state. Each line can be copied on its own; the
	Copy menu copies the lines shown, joins them into one line or downloads them.
-->
<script lang="ts">
	import ListViewIcon from '@hugeicons/core-free-icons/ListViewIcon';
	import ContainerIcon from '@hugeicons/core-free-icons/ContainerIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import AlertCircleIcon from '@hugeicons/core-free-icons/AlertCircleIcon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowDown02Icon from '@hugeicons/core-free-icons/ArrowDown02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import ComputerTerminal01Icon from '@hugeicons/core-free-icons/ComputerTerminal01Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Download04Icon from '@hugeicons/core-free-icons/Download04Icon';
	import FilterHorizontalIcon from '@hugeicons/core-free-icons/FilterHorizontalIcon';
	import PauseIcon from '@hugeicons/core-free-icons/PauseIcon';
	import PlayIcon from '@hugeicons/core-free-icons/PlayIcon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import TextWrapIcon from '@hugeicons/core-free-icons/TextWrapIcon';
	import WifiDisconnected01Icon from '@hugeicons/core-free-icons/WifiDisconnected01Icon';
	import { page } from '$app/state';
	import { tick } from 'svelte';
	import { toast } from 'svelte-sonner';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import Segmented from '#lib/components/segmented.svelte';
	import { copyText } from '#lib/clipboard.ts';
	import fetcher from '#lib/fetcher/index.ts';
	import l from '#lib/localization/index.ts';
	import appsService from '#lib/resources/apps.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import { LogsConnection } from '#lib/runtime-logs.svelte.ts';
	import {
		DEFAULT_TAIL,
		LineBuffer,
		TAIL_CHOICES,
		copyAll,
		copyOneLine,
		displayTime,
		formatLine,
		highlightParts,
		logFileName,
		matches,
		parseSearch,
		toLogFile,
		type ServiceState,
		type ViewLine
	} from '#lib/runtime-logs.ts';
	import { cn } from '#lib/utils.js';
	import AppWorkspace, { type AppActions } from '../app-workspace.svelte';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	// ----- What to stream -----

	let environment = $state<Environment>(
		page.url.searchParams.get('environment') === 'staging' ? 'staging' : 'production'
	);
	let tail = $state<string>(String(DEFAULT_TAIL));

	// ----- The lines -----

	const buffer = new LineBuffer();
	let lines = $state.raw<ViewLine[]>([]);
	let dropped = $state(false);
	let connection = $state<LogsConnection>();

	// ----- View -----

	let service = $state('all');
	let query = $state('');
	let errorsOnly = $state(false);
	let showTimes = $state(true);
	let wrap = $state(false);
	let paused = $state(false);
	let frozen = $state.raw<ViewLine[]>([]);
	let newWhilePaused = $state(0);
	let filtersOpen = $state(false);

	// ----- Containers -----

	let services = $state.raw<ServiceState[]>([]);
	let servicesTimer: ReturnType<typeof setTimeout> | undefined;

	async function loadServices(env: Environment) {
		try {
			services = await fetcher.get<ServiceState[]>(`/api/v1/apps/${data.app.id}/services`, {
				params: { environment: env },
				cache: 'no-store'
			});
		} catch {
			services = [];
		}
	}

	function refreshServicesSoon() {
		clearTimeout(servicesTimer);
		servicesTimer = setTimeout(() => loadServices(environment), 500);
	}

	// A new stream for each environment and number of lines to load.
	$effect(() => {
		const env = environment;
		const count = Number(tail);

		buffer.clear();
		lines = [];
		dropped = false;
		paused = false;
		frozen = [];
		newWhilePaused = 0;

		const stream = new LogsConnection(
			`/api/v1/apps/${data.app.id}/logs?environment=${env}&tail=${count}`,
			{
				lines(batch) {
					const added = buffer.append(batch);
					if (added === 0) return;
					lines = buffer.lines;
					dropped = buffer.dropped;
					if (paused) newWhilePaused += added;
				},
				event: refreshServicesSoon
			}
		);

		connection = stream;
		stream.start();

		return () => stream.stop();
	});

	$effect(() => {
		const env = environment;
		void loadServices(env);
		const timer = setInterval(() => loadServices(env), 30_000);
		return () => {
			clearInterval(timer);
			clearTimeout(servicesTimer);
		};
	});

	// ----- Derived view -----

	const search = $derived(parseSearch(query));
	const source = $derived(paused ? frozen : lines);
	const shown = $derived(
		source.filter((line) =>
			matches(line, { service: service === 'all' ? undefined : service, search, errorsOnly })
		)
	);
	const knownServices = $derived(
		[...new Set([...services.map((s) => s.service), ...lines.map((line) => line.service)])]
			.filter(Boolean)
			.sort()
	);
	const showService = $derived(service === 'all' && knownServices.length > 1);
	const format = $derived({ times: showTimes, service: showService });
	const filtering = $derived(
		service !== 'all' || errorsOnly || search.include.length > 0 || search.exclude.length > 0
	);
	const allStopped = $derived(
		services.length > 0 && services.every((s) => s.state !== 'running' && s.state !== 'restarting')
	);
	const problem = $derived(connection?.problem);
	const status = $derived(connection?.status ?? 'connecting');

	const environmentItems = $derived<{ value: Environment; label: string }[]>([
		{ value: 'production', label: l.translate('deploy.environment.production') },
		{ value: 'staging', label: l.translate('deploy.environment.staging') }
	]);

	// ----- Height: the console fills what is left of the workspace -----

	let sectionElement = $state<HTMLElement>();
	let consoleHeight = $state(480);

	$effect(() => {
		const section = sectionElement;
		const main = section?.closest('main');
		if (!section || !main) return;

		// The toolbar and the container chips wrap, so the console's top moves.
		const fit = () => {
			const top =
				section.getBoundingClientRect().top - main.getBoundingClientRect().top + main.scrollTop;
			const bottomPadding = parseFloat(getComputedStyle(section.parentElement!).paddingBottom) || 0;
			consoleHeight = Math.max(360, main.clientHeight - top - Math.max(bottomPadding, 32));
		};

		fit();
		const observer = new ResizeObserver(fit);
		observer.observe(main);
		if (section.previousElementSibling) observer.observe(section.previousElementSibling);
		section.parentElement?.querySelectorAll(':scope > *').forEach((el) => observer.observe(el));
		return () => observer.disconnect();
	});

	// ----- Follow and pause -----

	let consoleElement = $state<HTMLDivElement>();
	let programmaticScroll = false;

	function atBottom() {
		const el = consoleElement;
		return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 24;
	}

	function pause() {
		if (paused) return;
		frozen = lines;
		newWhilePaused = 0;
		paused = true;
	}

	async function follow() {
		paused = false;
		frozen = [];
		newWhilePaused = 0;
		await tick();
		scrollToBottom();
	}

	function scrollToBottom() {
		if (!consoleElement) return;
		programmaticScroll = true;
		consoleElement.scrollTop = consoleElement.scrollHeight;
	}

	function onConsoleScroll() {
		if (programmaticScroll) {
			programmaticScroll = false;
			return;
		}
		// Scrolling up freezes the view; scrolling back down follows again.
		if (!atBottom()) pause();
		else if (paused && newWhilePaused === 0) void follow();
	}

	// Keep the newest line in view while following.
	$effect(() => {
		void shown;
		if (!paused) scrollToBottom();
	});

	function clearView() {
		buffer.clear();
		lines = [];
		dropped = false;
		frozen = [];
		newWhilePaused = 0;
	}

	function showEverything() {
		query = '';
		errorsOnly = false;
		service = 'all';
	}

	// ----- Copy and download -----

	let fallbackText = $state('');
	let fallbackOpen = $state(false);
	let fallbackArea = $state<HTMLTextAreaElement>();

	async function copy(text: string, success: string) {
		if (await copyText(text)) {
			toast.success(success);
			return;
		}
		// Plain HTTP and an old method refused too: let the reader copy it themselves.
		fallbackText = text;
		fallbackOpen = true;
		await tick();
		fallbackArea?.focus();
		fallbackArea?.select();
	}

	const copyLine = (line: ViewLine) =>
		copy(formatLine(line, format), l.translate('logs.copied.line'));
	const copyShown = () =>
		copy(copyAll(shown, format), l.translate('logs.copied.all', [shown.length]));
	const copyJoined = () =>
		copy(copyOneLine(shown, format), l.translate('logs.copied.one_line', [shown.length]));

	function download() {
		const url = URL.createObjectURL(new Blob([toLogFile(shown)], { type: 'text/plain' }));
		const link = document.createElement('a');
		link.href = url;
		link.download = logFileName(app.name, environment);
		link.click();
		setTimeout(() => URL.revokeObjectURL(url), 1000);
	}

	// ----- Keyboard: one row in the tab order, arrows move between rows -----

	let activeId = $state<number>();
	const activeRow = $derived(
		shown.some((line) => line.id === activeId) ? activeId : shown.at(-1)?.id
	);

	function rowKeydown(event: KeyboardEvent, index: number) {
		const moves: Record<string, number> = {
			ArrowDown: 1,
			ArrowUp: -1,
			PageDown: 20,
			PageUp: -20,
			Home: -Infinity,
			End: Infinity
		};
		if (event.key === 'Enter' || event.key === 'c') {
			event.preventDefault();
			void copyLine(shown[index]);
			return;
		}
		const step = moves[event.key];
		if (step === undefined) return;
		event.preventDefault();
		const next = Math.max(0, Math.min(shown.length - 1, index + step));
		activeId = shown[next].id;
		if (next < index) pause();
		void tick().then(() =>
			consoleElement
				?.querySelector<HTMLElement>(`[data-line="${shown[next].id}"]`)
				?.focus({ preventScroll: false })
		);
	}

	// ----- Container chips -----

	function stateLabel(s: ServiceState) {
		if (s.health === 'unhealthy') return l.translate('logs.health.unhealthy');
		switch (s.state) {
			case 'running':
				return l.translate('logs.state.running');
			case 'restarting':
				return l.translate('logs.state.restarting');
			case 'exited':
				return l.translate('logs.state.exited', [s.exit_code]);
			case 'paused':
				return l.translate('logs.state.paused');
			case 'created':
				return l.translate('logs.state.created');
			case 'removing':
				return l.translate('logs.state.removing');
			default:
				return l.translate('logs.state.dead');
		}
	}

	function stateTone(s: ServiceState) {
		if (s.health === 'unhealthy' || s.state === 'exited' || s.state === 'dead')
			return 'text-danger';
		if (s.state === 'restarting') return 'text-warning';
		if (s.state === 'running') return 'text-success';
		return 'text-muted-foreground';
	}

	// Docker's summary repeats the exit code for exited containers ("Exited (3) 12 minutes ago").
	const stateDetail = (s: ServiceState) => s.status.replace(/^Exited \(\d+\)\s*/, '');

	const healthTone: Record<string, string> = {
		healthy: 'text-success',
		unhealthy: 'text-danger',
		starting: 'text-muted-foreground'
	};

	const statusTone = $derived(
		status === 'live' && !paused
			? 'text-success'
			: status === 'live' || status === 'reconnecting'
				? 'text-warning'
				: 'text-muted-foreground'
	);
	const statusLabel = $derived(
		status === 'live'
			? l.translate(paused ? 'logs.status.paused' : 'logs.status.live')
			: l.translate(`logs.status.${status}`)
	);
</script>

{#snippet serviceSelect(id: string)}
	<Select.Root type="single" bind:value={service}>
		<Select.Trigger {id} class="w-full md:w-44" aria-label={l.translate('logs.service')}>
			{service === 'all' ? l.translate('logs.service.all') : service}
		</Select.Trigger>
		<Select.Content>
			<Select.Item value="all" label={l.translate('logs.service.all')}>
				<Icon icon={Layers01Icon} class="text-muted-foreground" />
				{l.translate('logs.service.all')}
			</Select.Item>
			{#each knownServices as name (name)}
				<Select.Item value={name} label={name}>
					<Icon icon={ContainerIcon} class="text-muted-foreground" />
					{name}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
{/snippet}

{#snippet tailSelect(id: string)}
	<Select.Root type="single" bind:value={tail}>
		<Select.Trigger {id} class="w-full md:w-40" aria-label={l.translate('logs.tail')}>
			{l.translate('logs.tail.option', [Number(tail)])}
		</Select.Trigger>
		<Select.Content>
			{#each TAIL_CHOICES as count (count)}
				{@const label = l.translate('logs.tail.option', [count])}
				<Select.Item value={String(count)} {label}>
					<Icon icon={ListViewIcon} class="text-muted-foreground" />
					{label}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
{/snippet}

{#snippet errorsSwitch(id: string)}
	<div class="flex h-9 items-center gap-2">
		<Switch {id} bind:checked={errorsOnly} />
		<Label for={id} class="text-sm font-normal whitespace-nowrap">
			{l.translate('logs.errors_only')}
		</Label>
	</div>
{/snippet}

{#snippet viewToggles()}
	<Button
		variant="outline"
		size="icon"
		aria-label={l.translate('logs.timestamps')}
		title={l.translate('logs.timestamps')}
		aria-pressed={showTimes}
		class={cn(showTimes && 'bg-accent-soft text-primary hover:bg-accent-soft')}
		onclick={() => (showTimes = !showTimes)}
	>
		<Icon icon={Clock01Icon} />
	</Button>
	<Button
		variant="outline"
		size="icon"
		aria-label={l.translate('logs.wrap')}
		title={l.translate('logs.wrap')}
		aria-pressed={wrap}
		class={cn(wrap && 'bg-accent-soft text-primary hover:bg-accent-soft')}
		onclick={() => (wrap = !wrap)}
	>
		<Icon icon={TextWrapIcon} />
	</Button>
	<Button
		variant="ghost"
		size="icon"
		aria-label={l.translate('logs.clear')}
		title={l.translate('logs.clear')}
		onclick={clearView}
	>
		<Icon icon={Delete02Icon} />
	</Button>
{/snippet}

{#snippet copyMenu()}
	<div class="flex">
		<Button
			variant="outline"
			class="rounded-r-none"
			disabled={shown.length === 0}
			onclick={copyShown}
		>
			<Icon icon={Copy01Icon} />
			<span class="max-md:sr-only">{l.translate('logs.copy')}</span>
		</Button>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						variant="outline"
						size="icon"
						class="ml-0.5 w-8 rounded-l-none data-[state=open]:bg-line-strong"
						aria-label={l.translate('logs.copy.more')}
						disabled={shown.length === 0}
					>
						<Icon icon={ArrowDown01Icon} />
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end" sideOffset={4} class="w-75 p-1">
				<DropdownMenu.Item class="items-start gap-2.5 px-2.5 py-2" onSelect={copyShown}>
					<Icon icon={Copy01Icon} class="mt-0.5 text-muted-foreground" />
					<span class="flex flex-col gap-0.5">
						<span>{l.translate('logs.copy.all')}</span>
						<span class="text-xs text-muted-foreground">
							{l.translate('logs.copy.all.description', [shown.length])}
						</span>
					</span>
				</DropdownMenu.Item>
				<DropdownMenu.Item class="items-start gap-2.5 px-2.5 py-2" onSelect={copyJoined}>
					<Icon icon={TextWrapIcon} class="mt-0.5 text-muted-foreground" />
					<span class="flex flex-col gap-0.5">
						<span>{l.translate('logs.copy.one_line')}</span>
						<span class="text-xs text-muted-foreground">
							{l.translate('logs.copy.one_line.description')}
						</span>
					</span>
				</DropdownMenu.Item>
				<DropdownMenu.Item class="items-start gap-2.5 px-2.5 py-2" onSelect={download}>
					<Icon icon={Download04Icon} class="mt-0.5 text-muted-foreground" />
					<span class="flex flex-col gap-0.5">
						<span>{l.translate('logs.download')}</span>
						<span class="truncate text-xs text-muted-foreground">
							{logFileName(app.name, environment)}
						</span>
					</span>
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
{/snippet}

{#snippet problemState(actions: AppActions)}
	{#if problem === 'no_containers' || problem === 'not_found'}
		<EmptyState
			icon={ComputerTerminal01Icon}
			title={l.translate('logs.never.title')}
			description={l.translate('logs.never.description', [environment])}
		>
			{#snippet action()}
				<Button onclick={() => actions.quickDeploy(environment)}>
					{l.translate('logs.never.action', [environment])}
				</Button>
			{/snippet}
		</EmptyState>
	{:else if problem === 'too_many_streams'}
		<EmptyState
			icon={Alert02Icon}
			title={l.translate('logs.too_many.title')}
			description={l.translate('logs.too_many.description')}
		>
			{#snippet action()}
				<Button variant="outline" onclick={() => connection?.retry()}>
					<Icon icon={RefreshIcon} />
					{l.translate('logs.retry')}
				</Button>
			{/snippet}
		</EmptyState>
	{:else if problem === 'logs_unavailable'}
		<EmptyState
			icon={AlertCircleIcon}
			title={l.translate('logs.unavailable.title')}
			description={l.translate('logs.unavailable.description')}
		>
			{#snippet action()}
				<a
					class="text-sm font-semibold text-primary"
					href="https://docs.docker.com/engine/logging/configure/"
					target="_blank"
					rel="noreferrer"
				>
					{l.translate('logs.unavailable.link')}
				</a>
			{/snippet}
		</EmptyState>
	{:else}
		<EmptyState
			icon={AlertCircleIcon}
			title={l.translate('logs.failed.title')}
			description={l.translate('logs.failed.description')}
		>
			{#snippet action()}
				<Button variant="outline" onclick={() => connection?.retry()}>
					<Icon icon={RefreshIcon} />
					{l.translate('logs.retry')}
				</Button>
			{/snippet}
		</EmptyState>
	{/if}
{/snippet}

<AppWorkspace {app} apiKey={data.user.api_key} tab="logs">
	{#snippet children(actions: AppActions)}
		<div class="flex flex-col gap-3">
			<!-- Toolbar: filters on the left, live state and actions on the right; they wrap. -->
			<div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5">
				<div class="flex min-w-0 flex-wrap items-center gap-2 max-md:w-full">
					<span id="logs-environment" class="sr-only">{l.translate('logs.environment')}</span>
					<Segmented
						bind:value={environment}
						items={environmentItems}
						labelledby="logs-environment"
						class="max-md:w-full md:w-60"
					/>
					<div class="max-md:hidden">{@render serviceSelect('logs-service')}</div>
					<div class="max-md:hidden">{@render tailSelect('logs-tail')}</div>
					<div class="relative max-md:min-w-0 max-md:flex-1">
						<Label for="logs-search" class="sr-only">{l.translate('logs.search')}</Label>
						<Icon
							icon={Search01Icon}
							class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
						/>
						<Input
							id="logs-search"
							type="search"
							bind:value={query}
							placeholder={l.translate('logs.search.placeholder')}
							class={cn('pl-9 md:w-64', filtering && 'pr-20')}
						/>
						{#if filtering}
							<span
								class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-xs text-muted-foreground tabular-nums"
							>
								{l.translate('logs.search.count', [shown.length, source.length])}
							</span>
						{/if}
					</div>
					<div class="max-md:hidden">{@render errorsSwitch('logs-errors')}</div>
					<Button
						variant="outline"
						size="icon"
						class="md:hidden"
						aria-label={l.translate('logs.filters')}
						onclick={() => (filtersOpen = true)}
					>
						<Icon icon={FilterHorizontalIcon} />
					</Button>
				</div>

				<div class="flex flex-wrap items-center gap-2 max-md:w-full max-md:justify-between">
					<span
						role="status"
						class={cn('flex items-center gap-2 px-1 text-sm font-semibold', statusTone)}
					>
						<span class="size-2 rounded-full bg-current"></span>
						{statusLabel}
					</span>
					<div class="flex items-center gap-2">
						{#if paused}
							<Button onclick={follow}>
								<Icon icon={PlayIcon} />
								{l.translate('logs.follow')}
							</Button>
						{:else}
							<Button variant="outline" onclick={pause} disabled={status !== 'live'}>
								<Icon icon={PauseIcon} />
								{l.translate('logs.pause')}
							</Button>
						{/if}
						<div class="flex items-center gap-2 max-md:hidden">{@render viewToggles()}</div>
						{@render copyMenu()}
					</div>
				</div>
			</div>

			<!-- One entry per container (dot, name, state; no chip box), kept current by the stream's state events. -->
			{#if services.length > 0}
				<ul
					aria-label={l.translate('logs.containers')}
					class="flex gap-x-5 gap-y-1 max-md:-mx-1 max-md:overflow-x-auto max-md:px-1 md:flex-wrap"
				>
					{#each services as s (s.container)}
						<li class="flex min-h-8 items-center gap-2 text-label whitespace-nowrap">
							<span class={cn('size-2 rounded-full bg-current', stateTone(s))}></span>
							<strong class="font-semibold">{s.service}</strong>
							<span class={cn('font-semibold', stateTone(s))}>{stateLabel(s)}</span>
							{#if stateDetail(s)}
								<span class="text-muted-foreground">{stateDetail(s)}</span>
							{/if}
							{#if s.restart_count > 0}
								<span class="flex items-center gap-1 text-warning">
									<Icon icon={RefreshIcon} class="size-3.5" />
									{l.translate('logs.restarts', [s.restart_count])}
								</span>
							{/if}
							{#if s.health && s.health !== 'unhealthy'}
								<span class={healthTone[s.health] ?? 'text-muted-foreground'}>
									{l.translate(`logs.health.${s.health}` as 'logs.health.healthy')}
								</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			<section
				bind:this={sectionElement}
				aria-label={l.translate('detail.tab.logs')}
				style:height="{consoleHeight}px"
				class="relative flex flex-col overflow-hidden rounded-md bg-sunken"
			>
				{#if status === 'reconnecting'}
					<div
						role="status"
						class="m-2.5 mb-0 flex items-center gap-2.5 rounded-md bg-warning-soft px-3 py-2.5 text-label"
					>
						<Icon icon={WifiDisconnected01Icon} class="size-4.5 text-warning" />
						{l.translate('logs.reconnecting')}
					</div>
				{/if}

				{#if problem}
					<div class="flex flex-1 items-center justify-center p-6">
						{@render problemState(actions)}
					</div>
				{:else if status === 'connecting' && source.length === 0}
					<div class="flex flex-1 flex-col items-center justify-center gap-3">
						<span
							aria-hidden="true"
							class="size-6.5 animate-spin rounded-full border-2 border-line border-t-primary"
						></span>
						<p class="text-sm text-muted-foreground">{l.translate('logs.connecting')}</p>
					</div>
				{:else}
					<!-- The console scrolls with the keyboard when empty. -->
					<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
					<div
						bind:this={consoleElement}
						onscroll={onConsoleScroll}
						role="list"
						aria-label={l.translate('detail.tab.logs')}
						tabindex={shown.length === 0 ? 0 : -1}
						class="min-h-0 flex-1 overflow-auto px-2 py-2.5 font-mono text-xs/5"
					>
						{#if dropped}
							<p class="mb-1.5 px-3 font-sans text-xs text-muted-foreground">
								{l.translate('logs.dropped')}
							</p>
						{/if}

						{#each shown as line, index (line.id)}
							<!-- One row is in the tab order; arrows move between rows, Enter or C copies the row. -->
							<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
							<div
								role="listitem"
								data-line={line.id}
								tabindex={line.id === activeRow ? 0 : -1}
								onkeydown={(event) => rowKeydown(event, index)}
								onfocus={() => (activeId = line.id)}
								class="group relative flex items-start gap-3 rounded-sm px-3 py-px outline-none [contain-intrinsic-size:auto_20px] [content-visibility:auto] hover:bg-raised focus-visible:bg-raised focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset max-md:flex-col max-md:gap-0 max-md:py-1.5"
							>
								{#if showTimes}
									<span class="w-23 flex-none text-faint max-md:text-xs/4"
										>{displayTime(line.time)}</span
									>
								{/if}
								{#if showService}
									<span
										class="w-16 flex-none truncate text-muted-foreground max-md:absolute max-md:top-1.5 max-md:left-27 max-md:text-xs/4"
										>{line.service}</span
									>
								{/if}
								<span class="flex min-w-0 flex-1 gap-2">
									<span
										class={cn(
											'min-w-0',
											wrap
												? 'break-all whitespace-pre-wrap'
												: 'truncate whitespace-pre max-md:break-all max-md:whitespace-pre-wrap',
											line.error ? 'text-danger' : 'text-ink'
										)}
									>
										{#each highlightParts(line.text, search.include) as part, i (i)}
											{#if part.match}<mark class="rounded-xs bg-accent-soft px-px text-ink"
													>{part.text}</mark
												>{:else}{part.text}{/if}
										{/each}
										{#if line.truncated}
											<span class="font-sans text-xs text-muted-foreground">
												· {l.translate('logs.truncated')}</span
											>
										{/if}
									</span>
									{#if line.stream === 'stderr'}
										<span
											class="mt-0.5 flex h-4 flex-none items-center rounded-sm bg-raised px-1.5 font-sans text-xs font-semibold text-muted-foreground group-hover:bg-line-strong"
											>stderr</span
										>
									{/if}
								</span>
								<button
									type="button"
									tabindex="-1"
									aria-label={l.translate('logs.copy_line')}
									title={l.translate('logs.copy_line')}
									class="-my-px -mr-1 flex h-5.5 w-6.5 flex-none items-center justify-center rounded-sm text-ink opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-line-strong max-md:absolute max-md:top-1 max-md:right-1"
									onclick={() => copyLine(line)}
								>
									<Icon icon={Copy01Icon} class="size-3.5" />
								</button>
							</div>
						{/each}

						{#if source.length > 0 && shown.length === 0}
							<div class="flex h-full items-center justify-center p-6 font-sans">
								<EmptyState
									icon={Search01Icon}
									title={l.translate('logs.no_match.title')}
									description={l.translate('logs.no_match.description')}
								>
									{#snippet action()}
										<Button variant="outline" onclick={showEverything}>
											{l.translate('logs.no_match.action')}
										</Button>
									{/snippet}
								</EmptyState>
							</div>
						{:else if source.length === 0}
							<p class="px-3 py-2 font-sans text-sm text-muted-foreground">
								{l.translate('logs.empty')}
							</p>
						{/if}

						{#if allStopped && status === 'live'}
							<div class="mx-1 mt-3 flex flex-col items-center gap-2 p-3.5 text-center font-sans">
								<p class="text-sm font-semibold">{l.translate('logs.stopped.title')}</p>
								<p class="text-label text-muted-foreground">
									{l.translate('logs.stopped.description')}
								</p>
								<Button
									variant="outline"
									size="sm"
									onclick={() => actions.quickDeploy(environment)}
								>
									<Icon icon={RefreshIcon} />
									{l.translate('logs.redeploy')}
								</Button>
							</div>
						{/if}
					</div>

					{#if paused && newWhilePaused > 0}
						<Button class="absolute bottom-11 left-1/2 -translate-x-1/2 shadow-lg" onclick={follow}>
							<Icon icon={ArrowDown02Icon} />
							{l.translate('logs.new_lines', [newWhilePaused])}
						</Button>
					{/if}
				{/if}

				<div
					class="flex h-8 flex-none items-center justify-between gap-4 px-4 text-xs text-muted-foreground"
				>
					<span class="tabular-nums">
						{#if problem}
							<!-- Nothing is followed. -->
						{:else if paused}
							{l.translate('logs.footer.paused', [shown.length, source.length])}
						{:else if filtering}
							{l.translate('logs.footer.filtered', [shown.length, source.length])}
						{:else}
							{l.translate('logs.footer.following', [source.length])}
						{/if}
					</span>
					<span class="truncate max-md:hidden">{l.translate('logs.footer.source')}</span>
				</div>
			</section>
		</div>
	{/snippet}
</AppWorkspace>

<!-- Phone: the filters and view switches move into a sheet. -->
<Sheet.Root bind:open={filtersOpen}>
	<Sheet.Content side="bottom" class="gap-4 p-4">
		<Sheet.Header class="p-0">
			<Sheet.Title>{l.translate('logs.filters')}</Sheet.Title>
		</Sheet.Header>
		<div class="flex flex-col gap-1">
			<Label for="logs-service-phone">{l.translate('logs.service')}</Label>
			{@render serviceSelect('logs-service-phone')}
		</div>
		<div class="flex flex-col gap-1">
			<Label for="logs-tail-phone">{l.translate('logs.tail')}</Label>
			{@render tailSelect('logs-tail-phone')}
		</div>
		{@render errorsSwitch('logs-errors-phone')}
		<div class="flex gap-2">{@render viewToggles()}</div>
	</Sheet.Content>
</Sheet.Root>

<!-- Copy over plain HTTP when the browser refused both ways: the text is selected. -->
<Dialog.Root bind:open={fallbackOpen}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{l.translate('logs.fallback.title')}</Dialog.Title>
			<Dialog.Description>{l.translate('logs.fallback.description')}</Dialog.Description>
		</Dialog.Header>
		<textarea
			bind:this={fallbackArea}
			readonly
			value={fallbackText}
			aria-label={l.translate('logs.copy')}
			class="h-40 w-full resize-none rounded-md border border-input bg-transparent p-2.5 font-mono text-xs outline-none focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-inset"
		></textarea>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (fallbackOpen = false)}>
				{l.translate('logs.fallback.done')}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
