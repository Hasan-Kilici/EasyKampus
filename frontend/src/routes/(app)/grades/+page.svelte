<script lang="ts">
	import { Download, History } from '@lucide/svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { idMap, lookup } from '$lib/format';
	import { GRADE_TYPE_OPTS, type Field } from '$lib/options';
	import { input, ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let term = $state('');
	let type = $state('');
	let exportError = $state('');
	const courses = resource(() => api.courses.list().catch(() => [] as Rec[]));
	const names = $derived(idMap(courses.data ?? []));
	const m = $derived(session.canManage);

	const fields: Field[] = [
		{ name: 'score', label: 'Puan', type: 'number', required: true },
		{ name: 'note', label: 'Not' }
	];
	async function exportCsv() {
		exportError = '';
		try { await api.grades.export(); } catch (e) { exportError = errorMessage(e); }
	}
</script>

<CrudPage
	title="Notlar"
	subtitle="Sınav, ödev ve proje notların."
	columns={[{ key: 'course_id', label: 'Ders', format: (r) => lookup(names, r.course_id) }, { key: 'type', label: 'Tür' }, { key: 'score', label: 'Puan' }, { key: 'term', label: 'Dönem' }, { key: 'note', label: 'Not' }, { key: 'created_at', label: 'Tarih' }]}
	{fields}
	load={() => api.grades.list({ term, type })}
	update={m ? (r, v) => api.grades.update(String(r.id), v) : undefined}
	remove={m ? (r) => api.grades.remove(String(r.id)) : undefined}
>
	{#snippet headerExtra()}
		<input class="{input} !h-11 w-36" placeholder="Dönem" aria-label="Dönem filtresi" bind:value={term} />
		<select class="{input} !h-11 w-36" aria-label="Tür filtresi" bind:value={type}>
			<option value="">Tüm türler</option>
			{#each GRADE_TYPE_OPTS as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
		</select>
		{#if m}<button class={ghost} onclick={exportCsv}><Download class="size-4" />CSV</button>{/if}
		{#if exportError}<span class="text-sm font-semibold text-red-600" role="alert">{exportError}</span>{/if}
	{/snippet}
	{#snippet extra(row)}
		{#if m}<a class={ghost} href="/grades/{row.id}" aria-label="Değişiklik geçmişi"><History class="size-4" /></a>{/if}
	{/snippet}
</CrudPage>
