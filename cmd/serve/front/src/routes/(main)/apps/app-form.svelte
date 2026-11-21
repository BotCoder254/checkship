<!--
	Application settings (DESIGN.md §11.6): two columns, the section title and one sentence on the
	left, the fields on the right. Each environment is one row (target, Variables); its variables
	open in a drawer. New application: the page header is inside the form so its submit button
	submits. Settings (`embedded`): the application header and tabs come from AppWorkspace, so the
	same header shows on every tab, and the actions sit at the end of the form instead.
-->
<script lang="ts">
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import PencilEdit02Icon from '@hugeicons/core-free-icons/PencilEdit02Icon';
	import type { Snippet } from 'svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Collapsible from '#lib/components/ui/collapsible/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, {
		fromServiceVariablesRecord,
		toServiceVariablesRecord,
		type AppDetail,
		type CreateApp,
		type CreateAppDataEnvironmentConfig,
		type ServiceVariables,
		type UpdateApp
	} from '#lib/resources/apps.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import type { Target } from '#lib/resources/targets.ts';
	import { cn } from '#lib/utils.js';
	import EnvVarsDrawer from './env-vars-drawer.svelte';
	import DomainsDrawer from './domains-drawer.svelte';
	import { domainChips, worstTone, type ChipTone } from '#lib/hostname.ts';
	import type { DomainsCheck } from '#lib/resources/apps.ts';

	type Props = {
		/** Page header, when the form is its own page (New application). */
		title?: string;
		crumbs?: { href: string; label: string }[];
		actions: Snippet<[boolean]>;
		/** Inside the application workspace: no header; the actions close the form. */
		embedded?: boolean;
		targets: Target[];
		disabled?: boolean;
	} & (
		| { initialData: AppDetail; handler: (data: UpdateApp) => Promise<unknown> }
		| { initialData?: undefined; handler: (data: CreateApp) => Promise<unknown> }
	);

	let props: Props = $props();
	// The form is created once per page; later prop changes don't reset what the person typed.
	// svelte-ignore state_referenced_locally
	const initialData = props.initialData;
	// svelte-ignore state_referenced_locally
	const targets = props.targets;

	let name = $state(initialData?.name ?? '');
	let useVersionControl = $state(!!initialData?.version_control);
	let url = $state(initialData?.version_control?.url ?? '');
	let token = $state(initialData?.version_control?.token ?? '');
	let env = $state<Record<Environment, { target: string; vars: ServiceVariables[] }>>({
		production: {
			target: initialData?.production.target.id ?? targets[0]?.id ?? '',
			vars: fromServiceVariablesRecord(initialData?.production.vars)
		},
		staging: {
			target: initialData?.staging.target.id ?? targets[0]?.id ?? '',
			vars: fromServiceVariablesRecord(initialData?.staging.vars)
		}
	});
	// What the server has, so the variables drawer can save one environment on its own.
	const savedTargets: Record<Environment, string | undefined> = {
		production: initialData?.production.target.id,
		staging: initialData?.staging.target.id
	};

	const environments: Environment[] = ['production', 'staging'];
	let varsOpen = $state(false);
	let varsEnvironment = $state<Environment>('production');

	// Custom domains (seelf#5): existing applications only, saved from their own drawer. `current`
	// is refreshed after a save so the list and the drawer show what the server has.
	let current = $state(initialData);
	let domainsOpen = $state(false);
	let domainsEnvironment = $state<Environment>('production');
	let checks = $state<Partial<Record<Environment, DomainsCheck>>>({});

	const domainsOf = (environment: Environment) =>
		Object.entries(current?.[environment].domains ?? {}).flatMap(([service, hostnames]) =>
			hostnames.map((hostname, index) => ({ service, hostname, primary: index === 0 }))
		);

	async function loadChecks() {
		if (!current) return;
		for (const environment of environments) {
			if (!domainsOf(environment).length || !current[environment].target.url) continue;
			try {
				checks[environment] = await service.checkDomains(current.id, environment);
			} catch {
				// The list shows "not checked yet"
			}
		}
	}

	$effect(() => {
		void loadChecks();
	});

	async function domainsSaved() {
		if (!current) return;
		current = await service.fetchById(current.id, { cache: 'no-store' }); // The effect checks again
	}

	const dotClass: Record<ChipTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		danger: 'bg-danger',
		muted: 'bg-faint'
	};

	function domainDot(environment: Environment, hostname: string) {
		const check = checks[environment];
		const status = check?.domains.find((d) => d.hostname === hostname);
		return dotClass[worstTone(domainChips(status, check?.https ?? false))];
	}

	function domainsSummary(environment: Environment) {
		const count = domainsOf(environment).length;
		return count ? l.translate('domains.summary', [count]) : l.translate('domains.summary.none');
	}
	let howOpen = $state(false);

	const targetLabel = (id: string) => {
		const t = targets.find((t) => t.id === id);
		return t ? `${t.name} · ${t.url ?? l.translate('target.manual_proxy')}` : '';
	};

	function config(environment: Environment): CreateAppDataEnvironmentConfig {
		const vars = env[environment].vars;
		return {
			target: env[environment].target,
			vars: vars.length ? toServiceVariablesRecord(vars) : undefined
		};
	}

	function summary(environment: Environment) {
		const record = toServiceVariablesRecord(env[environment].vars);
		const services = Object.keys(record).length;
		if (!services) return l.translate('vars.summary.none');
		const variables = Object.values(record).reduce((n, v) => n + Object.keys(v).length, 0);
		return l.translate('vars.summary', [services, variables]);
	}

	let moveVolumes = $state(true);

	const changed = (environment: Environment) =>
		!!initialData && initialData[environment].target.id !== env[environment].target;

	/** The URLs a service gets, for the "How checkship exposes services" panel. */
	function exposed(environment: Environment) {
		const target = targets.find((t) => t.id === env[environment].target);
		const app = name || '<app>';
		const suffix = environment === 'staging' ? '-staging' : '';
		try {
			const u = new URL(target?.url ?? '');
			return {
				main: `${u.protocol}//${app}${suffix}.${u.host}`,
				other: `${u.protocol}//<service>.${app}${suffix}.${u.host}`
			};
		} catch {
			return { main: `${app}${suffix}.<target>`, other: `<service>.${app}${suffix}.<target>` };
		}
	}

	async function saveVars(list: ServiceVariables[]) {
		const environment = varsEnvironment;
		const saved = savedTargets[environment];
		if (initialData && saved) {
			// An existing application: save this environment's variables now, on the target it has.
			const only = {
				target: saved,
				vars: list.length ? toServiceVariablesRecord(list) : undefined
			};
			await service.update(initialData.id, {
				version_control: undefined,
				production: environment === 'production' ? only : undefined,
				staging: environment === 'staging' ? only : undefined
			});
		}
		env[environment].vars = list;
	}

	const { submit, loading, errors } = submitter(() => {
		if (!initialData) {
			return props.handler({
				name,
				version_control: useVersionControl ? { url, token: token || undefined } : undefined,
				production: config('production'),
				staging: config('staging')
			} as CreateApp & UpdateApp);
		}

		return props.handler({
			move_volumes: moveVolumes,
			version_control: useVersionControl
				? {
						url,
						token: token !== (initialData.version_control?.token ?? '') ? token || null : undefined
					}
				: null,
			production: config('production'),
			staging: config('staging')
		} as CreateApp & UpdateApp);
	});

	const fieldError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($loading) return;
		submit().catch(() => {});
	}
</script>

<form class="flex flex-col" novalidate {onsubmit}>
	{#if !props.embedded}
		<PageHeader title={props.title ?? ''} crumbs={props.crumbs}>
			{#snippet actions()}
				{@render props.actions($loading)}
			{/snippet}
		</PageHeader>
	{/if}

	<fieldset class="flex flex-col gap-10" disabled={props.disabled}>
		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		{#if targets.length === 0}
			<Alert.Root variant="warning">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">{l.translate('app.no_targets')}</Alert.Title>
				<Alert.Description>
					<a href={routes.createTarget} class="font-semibold underline">
						{l.translate('settings.create_target')}
					</a>
				</Alert.Description>
			</Alert.Root>
		{/if}

		<SettingsSection
			id="settings-general"
			title={l.translate('settings.general')}
			description={initialData
				? l.translate('settings.general.edit')
				: l.translate('settings.general.new')}
		>
			<div class="flex flex-col gap-1.5">
				<Label for="app-name">{l.translate('name')}</Label>
				<Input
					id="app-name"
					bind:value={name}
					readonly={!!initialData}
					required
					autofocus={!initialData}
					class={cn(initialData && 'border-transparent bg-raised text-muted-foreground')}
					aria-invalid={!!fieldError('name') || undefined}
					aria-describedby="app-name-message"
				/>
				<FieldMessage id="app-name-message" error={fieldError('name')}>
					{#snippet help()}
						{#if !initialData}{l.translate('settings.name.help')}{/if}
					{/snippet}
				</FieldMessage>
			</div>

			{#if !initialData}
				<Collapsible.Root bind:open={howOpen} class="flex flex-col gap-2.5">
					<Collapsible.Trigger
						class="-ml-2 flex h-8 items-center gap-2 self-start rounded-md px-2 text-sm font-semibold text-primary outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
					>
						<Icon icon={Globe02Icon} class="size-4" />
						{l.translate('app.how')}
						<Icon
							icon={ArrowDown01Icon}
							class={cn('size-4 transition-transform', howOpen && 'rotate-180')}
						/>
					</Collapsible.Trigger>
					<Collapsible.Content class="flex flex-col gap-2">
						<p class="text-label">{l.translate('settings.how.intro')}</p>
						<dl class="grid gap-x-4 gap-y-1.5 text-label sm:grid-cols-[max-content_minmax(0,1fr)]">
							{#each environments as environment (environment)}
								{@const urls = exposed(environment)}
								<dt class="truncate font-mono">{urls.main}</dt>
								<dd class="text-muted-foreground">
									{l.translate(`settings.how.main.${environment}`)}
								</dd>
								<dt class="truncate font-mono">{urls.other}</dt>
								<dd class="text-muted-foreground">{l.translate('settings.how.other')}</dd>
							{/each}
						</dl>
					</Collapsible.Content>
				</Collapsible.Root>
			{/if}
		</SettingsSection>

		<SettingsSection
			id="settings-source"
			title={l.translate('app.vcs')}
			description={l.translate('settings.vcs.description')}
		>
			<SettingRow
				for="app-vcs"
				label={l.translate('settings.vcs.use')}
				description={l.translate('settings.vcs.use.help')}
			>
				<Switch id="app-vcs" bind:checked={useVersionControl} />
			</SettingRow>

			{#if useVersionControl}
				<div class="flex flex-col gap-1.5">
					<Label for="app-vcs-url">{l.translate('settings.vcs.url')}</Label>
					<Input
						id="app-vcs-url"
						type="url"
						bind:value={url}
						required
						class="font-mono text-label"
						placeholder="https://github.com/acme/app"
						aria-invalid={!!fieldError('version_control.url') || undefined}
						aria-describedby="app-vcs-url-message"
					/>
					<FieldMessage id="app-vcs-url-message" error={fieldError('version_control.url')} />
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="app-vcs-token">{l.translate('app.vcs.token')}</Label>
					<Input
						id="app-vcs-token"
						type="password"
						autocomplete="new-password"
						bind:value={token}
						aria-describedby="app-vcs-token-message"
					/>
					<FieldMessage id="app-vcs-token-message">
						{#snippet help()}{l.translate('settings.vcs.token.help')}{/snippet}
					</FieldMessage>
				</div>
			{/if}
		</SettingsSection>

		<SettingsSection
			id="settings-environments"
			title={l.translate('settings.environments')}
			description={l.translate('settings.environments.description')}
			explain={initialData ? l.translate('settings.domains.explain') : undefined}
		>
			<div class="flex flex-col gap-3">
				{#each environments as environment (environment)}
					<div
						class={cn(
							'grid gap-3 py-2 sm:items-center sm:gap-4',
							initialData
								? 'sm:grid-cols-[120px_minmax(0,1fr)]'
								: 'sm:grid-cols-[120px_minmax(0,1fr)_auto]'
						)}
					>
						<span class="flex flex-col gap-0.5">
							<span id="app-{environment}-label" class="text-sm font-semibold">
								{l.translate(`deploy.environment.${environment}`)}
							</span>
							<span
								class={cn(
									'text-xs',
									changed(environment) ? 'text-warning' : 'text-muted-foreground'
								)}
							>
								{changed(environment)
									? l.translate('app.environment.target.changed')
									: initialData
										? `${domainsSummary(environment)} · ${summary(environment)}`
										: summary(environment)}
							</span>
						</span>
						<!-- Existing app: the target, then Domains and Variables under it (a new app keeps one row) -->
						<div class={initialData ? 'flex min-w-0 flex-col gap-2' : 'contents'}>
							<Select.Root type="single" bind:value={env[environment].target}>
								<Select.Trigger
									aria-labelledby="app-{environment}-label"
									aria-invalid={!!fieldError(`${environment}.target`) || undefined}
									class="w-full min-w-0"
								>
									<span class="truncate">{targetLabel(env[environment].target)}</span>
								</Select.Trigger>
								<Select.Content>
									{#each targets as target (target.id)}
										<Select.Item value={target.id} label={target.name}>
											<Icon icon={ServerStack01Icon} class="text-muted-foreground" />
											{targetLabel(target.id)}
										</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
							<div class={initialData ? 'flex flex-wrap gap-2' : 'contents'}>
								{#if initialData}
									<Button
										variant="outline"
										onclick={() => {
											domainsEnvironment = environment;
											domainsOpen = true;
										}}
									>
										<Icon icon={Globe02Icon} />
										{l.translate('settings.domains')}
										{#if domainsOf(environment).length}
											<span
												class="ml-0.5 inline-grid h-5 min-w-5 place-items-center rounded-full bg-surface px-1.5 text-xs font-semibold"
												>{domainsOf(environment).length}</span
											>
										{/if}
									</Button>
								{/if}
								<Button
									variant="outline"
									onclick={() => {
										varsEnvironment = environment;
										varsOpen = true;
									}}
								>
									<Icon icon={PencilEdit02Icon} />
									{l.translate('settings.variables')}
								</Button>
							</div>
						</div>
					</div>
					{#if initialData && domainsOf(environment).length}
						<div class="-mt-2 flex flex-col gap-1 text-label sm:pl-[136px]">
							<span class="text-xs text-muted-foreground">
								{l.translate('settings.domains.list', [
									l.translate(`deploy.environment.${environment}`)
								])}
							</span>
							<ul class="flex flex-col gap-1">
								{#each domainsOf(environment) as domain (domain.hostname)}
									<li class="flex min-w-0 items-center gap-2">
										<span
											aria-hidden="true"
											class={cn(
												'size-2 shrink-0 rounded-full',
												domainDot(environment, domain.hostname)
											)}
										></span>
										<span class="truncate font-mono">{domain.hostname}</span>
										<span class="shrink-0 text-xs text-muted-foreground">
											· {domain.service}{#if domain.primary && domain.service === Object.keys(current?.[environment].domains ?? {})[0]}
												· {l.translate('settings.domains.primary')}{/if}
										</span>
									</li>
								{/each}
							</ul>
						</div>
					{/if}
					{#if changed(environment) && initialData}
						<Alert.Root variant="warning">
							<Icon icon={Alert02Icon} />
							<Alert.Title class="font-semibold">
								{l.translate('settings.target_changed.title', [
									l.translate(`deploy.environment.${environment}`),
									initialData[environment].target.name,
									targets.find((t) => t.id === env[environment].target)?.name ?? ''
								])}
							</Alert.Title>
							<Alert.Description
								>{l.translate('settings.target_changed.description')}</Alert.Description
							>
						</Alert.Root>
						<SettingRow
							for="move-volumes-{environment}"
							label={l.translate('settings.move_volumes')}
							description={moveVolumes
								? l.translate('settings.move_volumes.on')
								: l.translate('settings.move_volumes.off')}
						>
							<Switch id="move-volumes-{environment}" bind:checked={moveVolumes} />
						</SettingRow>
					{/if}
				{/each}
			</div>
		</SettingsSection>
	</fieldset>

	{#if props.embedded}
		<!-- Destructive action on the left, Save on the right, under the last section's fields. -->
		<div class="mt-10 flex flex-wrap items-center justify-between gap-3 md:ml-[300px]">
			{@render props.actions($loading)}
		</div>
	{/if}
</form>

{#if current}
	<DomainsDrawer
		bind:open={domainsOpen}
		app={current}
		environment={domainsEnvironment}
		onsaved={domainsSaved}
	/>
{/if}

<EnvVarsDrawer
	bind:open={varsOpen}
	title={l.translate('vars.title', [l.translate(`deploy.environment.${varsEnvironment}`)])}
	services={env[varsEnvironment].vars}
	known={initialData?.latest_deployments[varsEnvironment]?.state.services?.map((s) => s.name)}
	saveLabel={initialData ? l.translate('vars.save') : l.translate('vars.done')}
	onsave={saveVars}
/>
