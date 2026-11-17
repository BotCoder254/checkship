<!--
	Empty-state drawings (DESIGN.md §11.5 Empty states): 14 SVG illustrations in one family, drawn
	in the app's own tokens so they follow light and dark with no second file. Objects are `surface`
	with a 2px outline, details `sunken` and `raised`, one accent marks the thing to do; good news
	is `success`, a problem `warning` or `danger`. A soft `sunken` floor shadow is part of the
	drawing, never a background. Decorative: hidden from assistive technology, the empty state's
	title and sentence say everything. The accent badge floats and the terminal cursor blinks, both
	off under reduced motion. In dark, outlines mix `line-strong` with 35% `muted` (derived from
	existing tokens) so they stay visible.
-->
<script lang="ts" module>
	export const illustrations = [
		'apps',
		'targets',
		'registries',
		'rocket',
		'backups',
		'queue',
		'search',
		'terminal',
		'logsWarn',
		'notFound',
		'error',
		'chart',
		'allClear',
		'volumes'
	] as const;
	export type IllustrationName = (typeof illustrations)[number];
</script>

<script lang="ts">
	import { cn } from '#lib/utils.js';

	type Props = { name: IllustrationName; class?: string };

	let { name, class: className }: Props = $props();
</script>

<svg
	class={cn('ill block h-auto w-full overflow-visible', className)}
	viewBox="0 0 240 160"
	aria-hidden="true"
	focusable="false"
>
	{#if name === 'apps'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="52" y="24" width="108" height="80" rx="9" />
		<circle class="r" cx="63" cy="35" r="2.6" />
		<circle class="r" cx="71" cy="35" r="2.6" />
		<circle class="r" cx="79" cy="35" r="2.6" />
		<rect class="o" x="74" y="44" width="114" height="92" rx="9" />
		<circle class="r" cx="86" cy="56" r="2.8" />
		<circle class="r" cx="95" cy="56" r="2.8" />
		<circle class="r" cx="104" cy="56" r="2.8" />
		<rect class="w" x="88" y="70" width="40" height="28" rx="5" />
		<rect class="acs" x="136" y="70" width="40" height="28" rx="5" />
		<rect class="w" x="88" y="104" width="40" height="22" rx="5" />
		<rect class="w" x="136" y="104" width="40" height="22" rx="5" />
		<g class="float"
			><circle class="ac" cx="190" cy="42" r="15" />
			<path class="sfl" d="M190 35v14M183 42h14" />
		</g>
		<path
			class="sp"
			transform="translate(40 68) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(206 104) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'targets'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="66" y="30" width="108" height="30" rx="7" />
		<rect class="o" x="66" y="66" width="108" height="30" rx="7" />
		<rect class="o" x="66" y="102" width="108" height="30" rx="7" />
		<circle class="ok" cx="81" cy="45" r="4" />
		<circle class="ok" cx="81" cy="81" r="4" />
		<circle class="r" cx="81" cy="117" r="4" />
		<rect class="w" x="93" y="41" width="50" height="8" rx="4" />
		<rect class="w" x="93" y="77" width="50" height="8" rx="4" />
		<rect class="w" x="93" y="113" width="50" height="8" rx="4" />
		<rect class="r" x="150" y="42" width="14" height="6" rx="3" />
		<rect class="r" x="150" y="78" width="14" height="6" rx="3" />
		<rect class="r" x="150" y="114" width="14" height="6" rx="3" />
		<g class="float"
			><path class="acl" d="M183 40a10 10 0 0 1 14 0" />
			<path class="acl" d="M177 33a19 19 0 0 1 26 0" />
			<circle class="ac" cx="190" cy="46" r="3" />
		</g>
		<path
			class="sp"
			transform="translate(44 56) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(200 120) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'registries'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="40" y="100" width="36" height="34" rx="3" />
		<path class="o" d="M40 100L50 90H86L76 100Z" />
		<path class="o" d="M76 100L86 90V124L76 134Z" />
		<rect class="o" x="84" y="66" width="78" height="68" rx="4" />
		<path class="o" d="M84 66L102 46H180L162 66Z" />
		<path class="o" d="M162 66L180 46V114L162 134Z" />
		<rect class="acs" x="114" y="67" width="18" height="66" />
		<path class="acs" d="M115 65L133 47H151L133 65Z" />
		<rect class="w" x="93" y="110" width="16" height="14" rx="2" />
		<g class="float"
			><circle class="ac" cx="196" cy="34" r="13" />
			<path class="sfl" d="M196 28v12M190 34h12" />
		</g>
		<path
			class="sp"
			transform="translate(56 62) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(206 96) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'rocket'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<circle class="w" cx="100" cy="139" r="10" />
		<circle class="w" cx="141" cy="140" r="9" />
		<circle class="w" cx="120" cy="143" r="8" />
		<g class="float">
			<path class="ac" d="M109 106C109 121 115 131 120 140C125 131 131 121 131 106Z" />
			<path class="acs" d="M114.5 106C114.5 116 117.5 123 120 128C122.5 123 125.5 116 125.5 106Z" />
			<path class="o" d="M98 80L80 106L98 102Z" />
			<path class="o" d="M142 80L160 106L142 102Z" />
			<path class="o" d="M120 16C141 33 147 62 143 98H97C93 62 99 33 120 16Z" />
			<rect class="r" x="106" y="98" width="28" height="9" rx="3" />
			<circle class="acs" cx="120" cy="56" r="12" />
			<circle class="st" cx="120" cy="56" r="12" />
		</g>
		<path
			class="sp"
			transform="translate(58 42) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(184 58) scale(0.9)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(172 24) scale(0.6)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'backups'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="86" y="30" width="34" height="22" rx="3" />
		<rect class="o" x="112" y="26" width="34" height="26" rx="3" />
		<rect class="o" x="70" y="68" width="100" height="66" rx="6" />
		<rect class="o" x="62" y="46" width="116" height="26" rx="6" />
		<rect class="r" x="104" y="86" width="32" height="9" rx="4.5" />
		<g class="float"
			><circle class="acs" cx="186" cy="40" r="18" />
			<path class="acl" d="M178 42a8.5 8.5 0 1 0 2.6-6.1" />
			<path class="acl" d="M179 31v6h6" />
		</g>
		<path
			class="sp"
			transform="translate(46 92) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(202 104) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'queue'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<path class="o" d="M54 90L72 56H168L186 90V126a8 8 0 0 1-8 8H62a8 8 0 0 1-8-8Z" />
		<path class="w" d="M56 92H96L102 104H138L144 92H184V126a6 6 0 0 1-6 6H62a6 6 0 0 1-6-6Z" />
		<path class="st" d="M56 92H96L102 104H138L144 92H184" />
		<g class="float"
			><circle class="okS" cx="120" cy="32" r="18" />
			<path class="okl" style="stroke-width:3.5" d="M112 32l5.5 5.5L129 26" />
		</g>
		<path
			class="okSp"
			transform="translate(66 40) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="okSp"
			transform="translate(176 36) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'search'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="44" y="30" width="122" height="100" rx="9" />
		<circle class="w" cx="63" cy="54" r="7" />
		<rect class="w" x="78" y="49" width="72" height="10" rx="5" />
		<circle class="w" cx="63" cy="80" r="7" />
		<rect class="w" x="78" y="75" width="56" height="10" rx="5" />
		<circle class="w" cx="63" cy="106" r="7" />
		<rect class="w" x="78" y="101" width="64" height="10" rx="5" />
		<g class="float"
			><circle class="lens" cx="166" cy="94" r="26" />
			<path class="acl" style="stroke-width:7" d="M185 113L204 132" />
			<path class="mu" d="M158 86l16 16M174 86l-16 16" />
		</g>
		<path
			class="sp"
			transform="translate(196 50) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'terminal'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="44" y="28" width="152" height="108" rx="10" />
		<circle class="r" cx="58" cy="42" r="3" />
		<circle class="r" cx="68" cy="42" r="3" />
		<circle class="r" cx="78" cy="42" r="3" />
		<rect class="w" x="60" y="60" width="62" height="8" rx="4" />
		<rect class="w" x="60" y="76" width="96" height="8" rx="4" />
		<path class="acl" d="M60 98l9 6.5-9 6.5" />
		<g class="blink"><rect class="ac" x="76" y="100" width="13" height="11" rx="2" /> </g>
		<path
			class="sp"
			transform="translate(210 56) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(32 108) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'logsWarn'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="44" y="28" width="152" height="108" rx="10" />
		<circle class="r" cx="58" cy="42" r="3" />
		<circle class="r" cx="68" cy="42" r="3" />
		<circle class="r" cx="78" cy="42" r="3" />
		<rect class="w" x="60" y="60" width="62" height="8" rx="4" />
		<rect class="w" x="60" y="76" width="96" height="8" rx="4" />
		<rect class="w" x="60" y="92" width="44" height="8" rx="4" />
		<g class="float"
			><path class="waT" d="M190 18L212 56H168Z" />
			<path class="wal" d="M190 32v11" />
			<circle class="wa" cx="190" cy="49.5" r="2" />
		</g>
	{:else if name === 'notFound'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<path class="o" d="M80 20H146L172 46V132a6 6 0 0 1-6 6H80a6 6 0 0 1-6-6V26a6 6 0 0 1 6-6Z" />
		<path class="ow" d="M146 20V40a6 6 0 0 0 6 6H172" />
		<path class="acl" style="stroke-width:6" d="M108 74a14 14 0 1 1 19 13c-4.5 2-7 5-7 9.5v4" />
		<circle class="ac" cx="120" cy="116" r="4.5" />
		<path
			class="sp"
			transform="translate(52 52) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="sp"
			transform="translate(194 90) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'error'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<path class="st" d="M42 84H16" />
		<rect class="o" x="42" y="68" width="46" height="32" rx="8" />
		<rect class="r" x="88" y="74" width="14" height="6" rx="2" />
		<rect class="r" x="88" y="88" width="14" height="6" rx="2" />
		<rect class="o" x="146" y="66" width="52" height="36" rx="8" />
		<rect class="w" x="153" y="75" width="9" height="6" rx="2" />
		<rect class="w" x="153" y="88" width="9" height="6" rx="2" />
		<path class="st" d="M198 84H224" />
		<g class="float"><path class="da" d="M126 52L113 82H124L117 108L139 74H128L135 52Z" /> </g>
		<path class="dal" d="M104 58l-6-6M146 58l6-6M125 44v-8" />
	{:else if name === 'chart'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<rect class="o" x="38" y="24" width="164" height="112" rx="10" />
		<rect class="w" x="54" y="40" width="50" height="8" rx="4" />
		<rect class="w" x="54" y="54" width="30" height="6" rx="3" />
		<path class="st" d="M56 120H184" />
		<rect class="dash" x="66" y="88" width="16" height="32" rx="3" />
		<rect class="dash" x="92" y="72" width="16" height="48" rx="3" />
		<rect class="dash" x="118" y="96" width="16" height="24" rx="3" />
		<rect class="dash" x="144" y="62" width="16" height="58" rx="3" />
		<g class="float"
			><path
				class="ac"
				transform="translate(176 42) scale(1.3)"
				d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
			/>
		</g><path
			class="sp"
			transform="translate(30 100) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'allClear'}
		<circle class="okR" cx="120" cy="80" r="62" />
		<circle class="okS" cx="120" cy="80" r="44" />
		<path class="okl" style="stroke-width:7" d="M99 81l14 14 28-29" />
		<path
			class="okSp"
			transform="translate(58 40) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="okSp"
			transform="translate(186 52) scale(0.8)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
		<path
			class="okSp"
			transform="translate(176 130) scale(0.7)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{:else if name === 'volumes'}
		<ellipse class="fl" cx="120" cy="148" rx="80" ry="6" />
		<path class="o" d="M72 38V116a48 14 0 0 0 96 0V38" />
		<path class="st" d="M72 64a48 14 0 0 0 96 0" />
		<path class="st" d="M72 90a48 14 0 0 0 96 0" />
		<ellipse class="o" cx="120" cy="38" rx="48" ry="14" />
		<ellipse class="w" cx="120" cy="38" rx="30" ry="7" />
		<circle class="ac" cx="154" cy="82" r="3.2" />
		<circle class="r" cx="154" cy="108" r="3.2" />
		<g class="float"
			><circle class="acs" cx="188" cy="44" r="14" />
			<path class="acl" d="M188 37v14M181 44h14" style="stroke-width:2.5" />
		</g>
		<path
			class="sp"
			transform="translate(46 70) scale(1)"
			d="M0 -6L1.6 -1.6L6 0L1.6 1.6L0 6L-1.6 1.6L-6 0L-1.6 -1.6Z"
		/>
	{/if}
</svg>

<style>
	.ill {
		--ill-line: var(--line-strong);
	}
	:global(.dark) .ill {
		--ill-line: color-mix(in oklab, var(--line-strong), var(--muted) 35%);
	}
	.ill :global(.o) {
		fill: var(--surface);
		stroke: var(--ill-line);
		stroke-width: 2;
		stroke-linejoin: round;
	}
	.ill :global(.ow) {
		fill: var(--sunken);
		stroke: var(--ill-line);
		stroke-width: 2;
		stroke-linejoin: round;
	}
	.ill :global(.w),
	.ill :global(.fl) {
		fill: var(--sunken);
	}
	.ill :global(.r) {
		fill: var(--raised);
	}
	.ill :global(.st) {
		fill: none;
		stroke: var(--ill-line);
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
	.ill :global(.dash) {
		fill: none;
		stroke: var(--ill-line);
		stroke-width: 2;
		stroke-dasharray: 4 5;
		stroke-linecap: round;
	}
	.ill :global(.ac) {
		fill: var(--accent);
	}
	.ill :global(.acs) {
		fill: var(--accent-soft);
	}
	.ill :global(.acl) {
		fill: none;
		stroke: var(--accent);
		stroke-width: 3;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
	.ill :global(.lens) {
		fill: var(--surface);
		stroke: var(--accent);
		stroke-width: 4;
	}
	.ill :global(.sfl) {
		fill: none;
		stroke: var(--surface);
		stroke-width: 3;
		stroke-linecap: round;
	}
	.ill :global(.sp) {
		fill: var(--accent);
		opacity: 0.55;
	}
	.ill :global(.mu) {
		fill: none;
		stroke: var(--muted);
		stroke-width: 2.5;
		stroke-linecap: round;
	}
	.ill :global(.ok) {
		fill: var(--success);
	}
	.ill :global(.okS) {
		fill: var(--success-soft);
	}
	.ill :global(.okl) {
		fill: none;
		stroke: var(--success);
		stroke-width: 5;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
	.ill :global(.okR) {
		fill: none;
		stroke: var(--success-soft);
		stroke-width: 2;
		stroke-dasharray: 3 6;
		stroke-linecap: round;
	}
	.ill :global(.okSp) {
		fill: var(--success);
		opacity: 0.6;
	}
	.ill :global(.waT) {
		fill: var(--warning-soft);
		stroke: var(--warning);
		stroke-width: 2.5;
		stroke-linejoin: round;
	}
	.ill :global(.wal) {
		fill: none;
		stroke: var(--warning);
		stroke-width: 3;
		stroke-linecap: round;
	}
	.ill :global(.wa) {
		fill: var(--warning);
	}
	.ill :global(.da) {
		fill: var(--danger);
	}
	.ill :global(.dal) {
		fill: none;
		stroke: var(--danger);
		stroke-width: 2.5;
		stroke-linecap: round;
	}
	@media (prefers-reduced-motion: no-preference) {
		.ill :global(.float) {
			animation: ill-float 5s ease-in-out infinite;
		}
		.ill :global(.blink) {
			animation: ill-blink 1.1s steps(1) infinite;
		}
	}
	@keyframes ill-float {
		50% {
			transform: translateY(-4px);
		}
	}
	@keyframes ill-blink {
		50% {
			opacity: 0;
		}
	}
</style>
