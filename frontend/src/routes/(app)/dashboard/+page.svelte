<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import QuickLinks from '$lib/components/QuickLinks.svelte';
	import Status from '$lib/components/Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { STATUS } from '$lib/format';
	import { card, ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const count = async (p: Promise<unknown[]>) => (await p).length;

	const res = resource(async () => {
		const stats: { label: string; value: number }[] = [];
		let perf: Rec | null = null;
		let perfError = '';
		if (session.isStudent) {
			stats.push({ label: 'Derslerim', value: await count(api.courses.list()) });
			try { perf = await api.students.me.performance(); } catch (e) { perfError = errorMessage(e); }
		} else if (session.isTeacher) {
			stats.push({ label: 'Sınıf', value: await count(api.classes.list()) });
			try { perf = await api.teachers.mePerformance(); } catch (e) { perfError = errorMessage(e); }
		} else {
			const [s, c, k, u] = await Promise.all([
				count(api.schools.list()), count(api.classes.list()), count(api.courses.list()), count(api.users.list())
			]);
			stats.push({ label: 'Okul', value: s }, { label: 'Sınıf', value: c }, { label: 'Ders', value: k }, { label: 'Kullanıcı', value: u });
		}
		return { stats, perf, perfError };
	});
	const result = $derived((res.data?.perf?.result ?? null) as Rec | null);
</script>

<PageHeader title={`Merhaba, ${session.user?.name ?? ''}`} subtitle="Bugünkü durumun bir bakışta." />

<QuickLinks />

<Status {res}>
	{#if res.data}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each res.data.stats as s (s.label)}<Stat label={s.label} value={s.value} />{/each}
			{#if result}
				<Stat label="Genel performans" value={Math.round(Number(result.overall_score))} hint={STATUS[String(result.overall_status)]} />
			{/if}
		</div>
		{#if res.data.perfError}<div class="{card} mt-4 font-semibold">{res.data.perfError}</div>{/if}
		{#if result}<a class="{ghost} mt-6" href="/performance">Performans ayrıntıları</a>{/if}
	{/if}
</Status>
