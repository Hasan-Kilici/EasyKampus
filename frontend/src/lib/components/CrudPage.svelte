<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Plus, Pencil, Trash2 } from '@lucide/svelte';
	import PageHeader from './PageHeader.svelte';
	import Status from './Status.svelte';
	import DataTable from './DataTable.svelte';
	import FormDialog from './FormDialog.svelte';
	import ConfirmDialog from './ConfirmDialog.svelte';
	import { resource } from '$lib/resource.svelte';
	import { errorMessage } from '$lib/api/client';
	import { solid, ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';
	import type { Field } from '$lib/options';

	type Col = { key: string; label: string; format?: (row: Rec) => string };
	let {
		title, subtitle, columns, fields = [], load, create, update, remove,
		createLabel = 'Yeni ekle', compact = false, emptyText, extra, headerExtra, summary
	}: {
		title: string; subtitle?: string; columns: Col[]; fields?: Field[];
		load: () => Promise<Rec[]>;
		create?: (v: Rec) => Promise<unknown>;
		update?: (row: Rec, v: Rec) => Promise<unknown>;
		remove?: (row: Rec) => Promise<unknown>;
		createLabel?: string; compact?: boolean; emptyText?: string;
		extra?: Snippet<[Rec, () => Promise<void>]>;
		headerExtra?: Snippet;
		summary?: Snippet;
	} = $props();

	const res = resource(() => load());
	const rows = $derived(res.data ?? []);
	let formOpen = $state(false);
	let editing = $state<Rec | null>(null);
	let delOpen = $state(false);
	let deleting = $state<Rec | null>(null);
	let actionError = $state('');

	const openCreate = () => { editing = null; formOpen = true; };
	const openEdit = (r: Rec) => { editing = r; formOpen = true; };
	const askDelete = (r: Rec) => { deleting = r; delOpen = true; };

	async function save(v: Rec) {
		if (editing) await update!(editing, v);
		else await create!(v);
		await res.reload();
	}
	async function confirmDelete() {
		actionError = '';
		try { await remove!(deleting!); await res.reload(); } catch (e) { actionError = errorMessage(e); }
	}
</script>

{#snippet bar()}
	{@render headerExtra?.()}
	{#if create}<button class={solid} onclick={openCreate}><Plus class="size-5" />{createLabel}</button>{/if}
{/snippet}

{#snippet actions(row: Rec)}
	{@render extra?.(row, res.reload)}
	{#if update}
		<button class="{ghost} !px-3" aria-label="Düzenle" onclick={() => openEdit(row)}><Pencil class="size-4" /></button>
	{/if}
	{#if remove}
		<button class="{ghost} !px-3" aria-label="Sil" onclick={() => askDelete(row)}><Trash2 class="size-4" /></button>
	{/if}
{/snippet}

{#if compact}
	<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
		<h2 class="text-xl font-bold">{title}</h2>
		<div class="flex flex-wrap items-center gap-2">{@render bar()}</div>
	</div>
{:else}
	<PageHeader {title} {subtitle} actions={bar} />
{/if}

{#if actionError}<p class="mb-3 font-semibold text-red-600" role="alert">{actionError}</p>{/if}

{@render summary?.()}

<Status {res} empty={!rows.length} {emptyText}>
	<DataTable {columns} {rows} actions={update || remove || extra ? actions : undefined} />
</Status>

{#if create || update}
	<FormDialog bind:open={formOpen} title={editing ? 'Düzenle' : createLabel} {fields} initial={editing ?? {}} isCreate={!editing} onsubmit={save} />
{/if}
{#if remove}
	<ConfirmDialog bind:open={delOpen} title="Bu kaydı sil?" text="Bu işlem geri alınamaz." onconfirm={confirmDelete} />
{/if}
