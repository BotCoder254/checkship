/**
 * Table columns (DESIGN.md §5 Tables): each table lists its columns once, with a set width so
 * columns never jump, a priority that decides which ones drop when the table gets narrow (the
 * table never scrolls sideways) and whether the Columns menu may hide it. What a person hides is
 * remembered per browser; storage that is blocked just falls back to every column shown.
 */
import type { IconData } from '#lib/components/icon.svelte';
import { cn } from '#lib/utils.js';

/** 1 always shows, 2 drops below 560px of table width, 3 drops below 760px. */
export type ColumnPriority = 1 | 2 | 3;

export type TableColumn<Id extends string = string> = {
	id: Id;
	/** One word (DESIGN.md §5). */
	label: string;
	/** Shown beside the column in the Columns menu. */
	icon: IconData;
	/** A Tailwind width class, such as `w-[18%]`. */
	width: string;
	priority?: ColumnPriority;
	/** Name and the main state: the Columns menu cannot hide them. */
	locked?: boolean;
	align?: 'right';
};

// Literal classes, so Tailwind sees them. The query is on the table's own width (table.svelte).
const dropBelow: Record<ColumnPriority, string> = {
	1: '',
	2: '@max-[560px]:hidden',
	3: '@max-[760px]:hidden'
};

const storageKey = (key: string) => `checkship:columns:${key}`;

function readHidden(key: string): string[] {
	try {
		const value = JSON.parse(localStorage.getItem(storageKey(key)) ?? '[]');
		return Array.isArray(value) ? value.filter((v) => typeof v === 'string') : [];
	} catch {
		return [];
	}
}

export class TableColumns<Id extends string> {
	readonly key: string;
	readonly all: readonly TableColumn<Id>[];
	#hidden = $state<string[]>([]);

	/** `key` names the table in storage; omit it for a table whose columns cannot be hidden. */
	constructor(columns: readonly TableColumn<Id>[], key = '') {
		this.all = columns;
		this.key = key;
		const locked: string[] = columns.filter((c) => c.locked).map((c) => c.id);
		this.#hidden = key ? readHidden(key).filter((id) => !locked.includes(id)) : [];
	}

	/** The columns shown, in order. */
	get shown(): TableColumn<Id>[] {
		return this.all.filter((c) => !this.#hidden.includes(c.id));
	}

	visible(id: Id): boolean {
		return !this.#hidden.includes(id);
	}

	setVisible(id: Id, show: boolean) {
		const column = this.all.find((c) => c.id === id);
		if (!column || column.locked) return;
		this.#hidden = show ? this.#hidden.filter((h) => h !== id) : [...this.#hidden, id];
		if (!this.key) return;
		try {
			localStorage.setItem(storageKey(this.key), JSON.stringify(this.#hidden));
		} catch {
			// Storage blocked: the choice lasts for this page only.
		}
	}

	/** Classes for a column's header cell: its width, when it drops, its alignment. */
	head(id: Id, className?: string): string {
		const column = this.all.find((c) => c.id === id)!;
		return cn(
			column.width,
			dropBelow[column.priority ?? 1],
			column.align === 'right' && 'text-right',
			className
		);
	}

	/** Classes for a body cell of the column: when it drops and its alignment. */
	cell(id: Id, className?: string): string {
		const column = this.all.find((c) => c.id === id)!;
		return cn(dropBelow[column.priority ?? 1], column.align === 'right' && 'text-right', className);
	}
}

/** The Actions column: the same 80px on every table (room for its one-word header), right-aligned. */
export const ACTIONS_HEAD = 'w-20 text-right';
export const ACTIONS_CELL = 'relative z-10 overflow-visible text-right';
