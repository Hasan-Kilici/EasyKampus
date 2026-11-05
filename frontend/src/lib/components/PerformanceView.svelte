<script lang="ts">
	import Stat from './Stat.svelte';
	import DataTable from './DataTable.svelte';
	import { STATUS, autoColumns } from '$lib/format';
	import { card } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let { data }: { data: Rec } = $props();

	const r = $derived((data?.result ?? data ?? {}) as Rec);
	const M: Record<string, string> = { activity: 'Aktivite', attendance: 'Devam', completion: 'Tamamlama', content: 'İçerik', quiz_metrics: 'Sınav' };
	const metrics = $derived(Object.keys(M).filter((k) => r[k] && typeof r[k] === 'object'));
	const lists = $derived(
		(['courses', 'classes', 'weak_areas'] as const).filter((k) => Array.isArray(r[k]) && (r[k] as unknown[]).length)
	);
	const L: Record<string, string> = { courses: 'Dersler', classes: 'Sınıflar', weak_areas: 'Zayıf konular' };
	const problems = $derived((r.problems ?? []) as Rec[]);
	const recs = $derived((r.recommendations ?? []) as Rec[]);
</script>

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
	<Stat label="Genel puan" value={Math.round(Number(r.overall_score ?? 0))} hint={STATUS[String(r.overall_status)] ?? ''} />
	{#each metrics as k}
		{@const m = r[k] as Rec}
		<Stat label={M[k]} value={Math.round(Number(m.score ?? 0))} hint={STATUS[String(m.status)] ?? ''} />
	{/each}
</div>

{#if problems.length || recs.length}
	<div class="mt-6 grid gap-4 lg:grid-cols-2">
		{#if problems.length}
			<div class={card}>
				<h2 class="mb-3 text-xl font-bold">Sorunlar</h2>
				<ul class="space-y-3">
					{#each problems as p}<li><span class="font-bold">{STATUS[String(p.severity)] ?? p.severity}:</span> {p.message}</li>{/each}
				</ul>
			</div>
		{/if}
		{#if recs.length}
			<div class={card}>
				<h2 class="mb-3 text-xl font-bold">Öneriler</h2>
				<ul class="space-y-3">
					{#each recs as x}<li><p class="font-bold">{x.title}</p><p class="text-zinc-600 dark:text-zinc-400">{x.message}</p></li>{/each}
				</ul>
			</div>
		{/if}
	</div>
{/if}

{#each lists as k}
	<h2 class="mt-8 mb-3 text-xl font-bold">{L[k]}</h2>
	<DataTable columns={autoColumns(r[k] as Rec[])} rows={r[k] as Rec[]} />
{/each}
