<script lang="ts">
	import type { Snippet } from 'svelte';
	import { fmt } from '$lib/format';
	import type { Rec } from '$lib/api/types';

	export type Col = { key: string; label: string; format?: (row: Rec) => string };

	let {
		columns,
		rows,
		actions,
		selectable = false,
		selected = $bindable<string[]>([]),
		rowKey = (row: Rec) => String(row.id ?? ''),
		cell
	}: {
		columns: Col[];
		rows: Rec[];
		actions?: Snippet<[Rec]>;
		selectable?: boolean;
		selected?: string[];
		rowKey?: (row: Rec) => string;
		cell?: Snippet<[Rec, Col]>;
	} = $props();

	const allSelected = $derived(rows.length > 0 && rows.every((row) => selected.includes(rowKey(row))));

	function toggleAll(event: Event) {
		const checked = (event.currentTarget as HTMLInputElement).checked;
		selected = checked ? rows.map(rowKey) : [];
	}

	function toggle(key: string, checked: boolean) {
		selected = checked ? [...new Set([...selected, key])] : selected.filter((value) => value !== key);
	}
</script>

<div class="overflow-x-auto rounded-lg border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
	<table class="w-full min-w-max text-left text-sm">
		<thead class="border-b border-zinc-200 bg-zinc-50 text-xs font-semibold tracking-wide text-zinc-600 uppercase dark:border-zinc-800 dark:bg-zinc-800/60 dark:text-zinc-300">
			<tr>
				{#if selectable}
					<th class="w-10 px-4 py-3">
						<input
							type="checkbox"
							aria-label="Tümünü seç"
							checked={allSelected}
							onchange={toggleAll}
						/>
					</th>
				{/if}
				{#each columns as c}<th class="px-4 py-3 font-semibold">{c.label}</th>{/each}
				{#if actions}<th class="px-4 py-3"><span class="sr-only">İşlemler</span></th>{/if}
			</tr>
		</thead>
		<tbody>
			{#each rows as row}
				{@const key = rowKey(row)}
				<tr class="border-b border-zinc-100 last:border-0 hover:bg-zinc-50/70 dark:border-zinc-800 dark:hover:bg-zinc-800/40">
					{#if selectable}
						<td class="px-4 py-3">
							<input
								type="checkbox"
								aria-label="Satırı seç"
								checked={selected.includes(key)}
								onchange={(event) => toggle(key, (event.currentTarget as HTMLInputElement).checked)}
							/>
						</td>
					{/if}
					{#each columns as c}
						<td class="px-4 py-3">
							{#if cell}{@render cell(row, c)}{:else if c.format}{c.format(row)}{:else}{fmt(c.key, row[c.key])}{/if}
						</td>
					{/each}
					{#if actions}
						<td class="px-4 py-2"><div class="flex items-center justify-end gap-1">{@render actions(row)}</div></td>
					{/if}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
