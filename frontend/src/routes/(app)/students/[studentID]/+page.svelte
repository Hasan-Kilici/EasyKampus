<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Tabs from '$lib/components/Tabs.svelte';
	import Status from '$lib/components/Status.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import PerformanceView from '$lib/components/PerformanceView.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { autoColumns, fmt } from '$lib/format';
	import { card, ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	// Öğretmen ve yönetici için öğrenci ayrıntısı. /users/:id yalnızca yöneticiye açıktır; bu sayfa
	// /students/:id/* uçlarını kullanır ve sunucu yalnızca kendi sınıfındaki öğrenciye izin verir.
	const id = $derived(page.params.studentID!);
	const profile = resource(() => api.student(id).profile() as Promise<Rec>);
	let tab = $state('performance');

	const tabs = [
		{ id: 'performance', label: 'Performans' }, { id: 'profile', label: 'Profil' }, { id: 'grades', label: 'Notlar' },
		{ id: 'attendance', label: 'Devamsızlık' }, { id: 'enrollments', label: 'Kayıtlar' }, { id: 'notes', label: 'Öğretmen notları' }
	];
	const s = $derived(api.student(id));
	const loaders: Record<string, () => Promise<Rec | Rec[]>> = {
		performance: () => s.performance(), profile: () => s.profile(), grades: () => s.grades(), attendance: () => s.attendance(),
		enrollments: () => s.enrollments(), notes: () => s.notes()
	};
	const data = resource<Rec | Rec[]>(() => loaders[tab]());
	const rows = $derived(Array.isArray(data.data) ? data.data : []);
</script>

<PageHeader title={`${profile.data?.name ?? ''} ${profile.data?.surname ?? ''}`.trim() || 'Öğrenci'} subtitle="Öğrenci">
	{#snippet actions()}<a class={ghost} href="/classes">Sınıflara dön</a>{/snippet}
</PageHeader>

<Tabs items={tabs} bind:value={tab} />

<Status res={data} empty={Array.isArray(data.data) && !rows.length}>
	{#if tab === 'performance'}
		{#if data.data && !Array.isArray(data.data)}<PerformanceView data={data.data} />{/if}
	{:else if Array.isArray(data.data)}
		<DataTable columns={autoColumns(rows)} {rows} />
	{:else if data.data}
		<dl class="{card} grid gap-3 sm:grid-cols-2">
			{#each Object.entries(data.data).filter(([, v]) => typeof v !== 'object') as [k, v] (k)}
				<div><dt class="text-sm font-semibold text-zinc-600 dark:text-zinc-400">{k}</dt><dd class="font-bold">{fmt(k, v)}</dd></div>
			{/each}
		</dl>
	{/if}
</Status>
