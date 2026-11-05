<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Tabs from '$lib/components/Tabs.svelte';
	import Status from '$lib/components/Status.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import PerformanceView from '$lib/components/PerformanceView.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { STATUS, autoColumns, fmt } from '$lib/format';
	import { card, ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const id = $derived(page.params.userID!);
	const user = resource(() => api.users.get(id) as unknown as Promise<Rec>);
	const role = $derived(String(user.data?.role ?? ''));
	let tab = $state('performance');

	const tabs = [
		{ id: 'performance', label: 'Performans' }, { id: 'profile', label: 'Profil' }, { id: 'grades', label: 'Notlar' },
		{ id: 'attendance', label: 'Devamsızlık' }, { id: 'enrollments', label: 'Kayıtlar' }, { id: 'lessons', label: 'Dersler' },
		{ id: 'notes', label: 'Notlar (öğretmen)' }, { id: 'activities', label: 'Aktiviteler' }, { id: 'daily', label: 'Günlük aktivite' }
	];
	const s = $derived(api.student(id));
	const loaders: Record<string, () => Promise<Rec | Rec[]>> = {
		performance: () => s.performance(), profile: () => s.profile(), grades: () => s.grades(), attendance: () => s.attendance(),
		enrollments: () => s.enrollments(), lessons: () => s.lessons(), notes: () => s.notes(), activities: () => s.activities(), daily: () => s.daily()
	};
	const data = resource<Rec | Rec[] | null>(async () =>
		role === 'student' ? loaders[tab]() : role === 'teacher' ? api.teachers.performance(id) : null
	);
	const rows = $derived(Array.isArray(data.data) ? data.data : []);
</script>

<PageHeader title={`${user.data?.name ?? ''} ${user.data?.surname ?? ''}`} subtitle={STATUS[role] ?? ''}>
	{#snippet actions()}<a class={ghost} href="/users">Kullanıcılara dön</a>{/snippet}
</PageHeader>

{#if role === 'student'}<Tabs items={tabs} bind:value={tab} />{/if}

{#if role === 'student' || role === 'teacher'}
	<Status res={data} empty={Array.isArray(data.data) && !rows.length}>
		{#if role === 'teacher' || tab === 'performance'}
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
{:else}
	<p class="text-zinc-600 dark:text-zinc-400">Bu rol için ek ayrıntı sayfası yok.</p>
{/if}
