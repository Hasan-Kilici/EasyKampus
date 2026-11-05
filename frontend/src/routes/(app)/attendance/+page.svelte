<script lang="ts">
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { idMap, lookup } from '$lib/format';
	import { ATTENDANCE_OPTS, type Field } from '$lib/options';
	import { card, input } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let status = $state('');
	let view = $state<'list' | 'summary'>('list');
	let from = $state('');
	let to = $state('');

	const courses = resource(() => api.courses.list().catch(() => [] as Rec[]));
	const names = $derived(idMap(courses.data ?? []));
	const m = $derived(session.canManage);

	const summaryRes = resource(() => {
		if (view !== 'summary') return Promise.resolve(null);
		const query: Record<string, string> = {};
		if (status) query.status = status;
		if (from) query.from = from;
		if (to) query.to = to;
		return api.attendance.summary(query);
	});

	const fields: Field[] = [
		{ name: 'status', label: 'Durum', type: 'select', options: ATTENDANCE_OPTS, required: true },
		{ name: 'note', label: 'Not' }
	];

	const query = $derived.by(() => {
		const q: Record<string, string> = {};
		if (status) q.status = status;
		if (view === 'list') {
			if (from) q.from = from;
			if (to) q.to = to;
		}
		return q;
	});
</script>

<CrudPage
	title="Devamsızlık"
	subtitle="Ders katılım kayıtlarını filtreleyebilir, günlük/haftalık/aylık görünümde özetleyebilirsiniz."
	columns={[{ key: 'date', label: 'Tarih' }, { key: 'course_id', label: 'Ders', format: (r) => lookup(names, r.course_id) }, { key: 'status', label: 'Durum' }, { key: 'note', label: 'Not' }]}
	{fields}
	load={() => api.attendance.list(query)}
	update={m ? (r, v) => api.attendance.update(String(r.id), { ...v, class_id: r.class_id, course_id: r.course_id, student_id: r.student_id, date: r.date }) : undefined}
	remove={m ? (r) => api.attendance.remove(String(r.id)) : undefined}
>
	{#snippet headerExtra()}
		<select class="{input} !h-11 w-40" aria-label="Görünüm" bind:value={view}>
			<option value="list">Kayıtlar</option>
			<option value="summary">Özet</option>
		</select>
		<input class="{input} !h-11 w-40" type="date" aria-label="Başlangıç tarihi" bind:value={from} />
		<input class="{input} !h-11 w-40" type="date" aria-label="Bitiş tarihi" bind:value={to} />
		<select class="{input} !h-11 w-44" aria-label="Durum filtresi" bind:value={status}>
			<option value="">Tüm durumlar</option>
			{#each ATTENDANCE_OPTS as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
		</select>
	{/snippet}

	{#snippet summary()}
		{#if view === 'summary'}
			<div class="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
				<div class={card}><p class="text-sm text-zinc-500">Toplam</p><p class="text-2xl font-bold">{summaryRes.data?.total ?? 0}</p></div>
				<div class={card}><p class="text-sm text-zinc-500">Var</p><p class="text-2xl font-bold">{summaryRes.data?.present ?? 0}</p></div>
				<div class={card}><p class="text-sm text-zinc-500">Yok</p><p class="text-2xl font-bold">{summaryRes.data?.absent ?? 0}</p></div>
				<div class={card}><p class="text-sm text-zinc-500">Geç</p><p class="text-2xl font-bold">{summaryRes.data?.late ?? 0}</p></div>
				<div class={card}><p class="text-sm text-zinc-500">Mazeretli</p><p class="text-2xl font-bold">{summaryRes.data?.excused ?? 0}</p></div>
				<div class={card}><p class="text-sm text-zinc-500">Katılım oranı</p><p class="text-2xl font-bold">%{Math.round((summaryRes.data?.rate ?? 0) * 100)}</p></div>
			</div>
			<div class="mb-6 overflow-x-auto rounded-lg border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
				<table class="w-full min-w-max text-left text-base">
					<thead class="border-b border-zinc-200 text-sm dark:border-zinc-800">
						<tr>
							<th class="px-4 py-3 font-bold">Tarih</th>
							<th class="px-4 py-3 font-bold">Toplam</th>
							<th class="px-4 py-3 font-bold">Var</th>
							<th class="px-4 py-3 font-bold">Yok</th>
							<th class="px-4 py-3 font-bold">Geç</th>
							<th class="px-4 py-3 font-bold">Mazeretli</th>
						</tr>
					</thead>
					<tbody>
						{#each summaryRes.data?.days ?? [] as day (day.date)}
							<tr class="border-b border-zinc-100 last:border-0 dark:border-zinc-800">
								<td class="px-4 py-3">{day.date}</td>
								<td class="px-4 py-3">{day.total}</td>
								<td class="px-4 py-3">{day.present}</td>
								<td class="px-4 py-3">{day.absent}</td>
								<td class="px-4 py-3">{day.late}</td>
								<td class="px-4 py-3">{day.excused}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/snippet}
</CrudPage>
