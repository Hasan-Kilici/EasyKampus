<script lang="ts">
	import { page } from '$app/state';
	import { ArrowRight } from '@lucide/svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { idMap } from '$lib/format';
	import type { Opt } from '$lib/options';
	import { input, ghost, solid } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let classId = $state(page.url.searchParams.get('class') ?? '');

	// Öğrenci: kayıtlı olduğu sınıflar; öğretmen/yönetici: sınıf listesi
	const classes = resource(async (): Promise<Opt[]> => {
		const all = await api.classes.list().catch(() => [] as Rec[]);
		const names = idMap(all);
		const ids = session.isStudent
			? [...new Set((await api.students.me.enrollments()).map((e) => String(e.class_id)))]
			: all.map((c) => String(c.id));
		return ids.map((id) => ({ value: id, label: names.get(id) ?? id.slice(0, 8) }));
	});
	$effect(() => {
		if (!classId && classes.data?.length) classId = classes.data[0].value;
	});

	const m = $derived(session.canManage);
</script>

{#if classes.data?.length}
	<div class="mb-4 max-w-xs">
		<label class="mb-2 block text-sm font-semibold" for="class-select">Sınıf</label>
		<select id="class-select" class={input} bind:value={classId}>
			{#each classes.data as c (c.value)}<option value={c.value}>{c.label}</option>{/each}
		</select>
	</div>
{/if}

{#if classId}
	{#if m}
		<div class="mb-4 flex justify-end"><a class={solid} href="/quizzes/new?class={classId}">Yeni sınav</a></div>
	{/if}
	{#key classId}
		<CrudPage
			title="Sınavlar"
			subtitle={m ? 'Sınavları oluştur, yayına gönder, sonuçları gör.' : 'Sınava gir, sonuçlarını gör.'}
			columns={[{ key: 'name', label: 'Sınav' }, { key: 'open_at', label: 'Açılış' }, { key: 'close_at', label: 'Kapanış' }, { key: 'duration_minutes', label: 'Süre (dk)' }, { key: 'max_attempts', label: 'Hak' }, { key: 'state', label: 'Durum' }]}
			fields={[]}
			load={() => api.class(classId).quizzes.list()}
			remove={m ? (r) => api.class(classId).quizzes.remove(String(r.id)) : undefined}
		>
			{#snippet extra(row)}
				{#if m}<a class={ghost} href="/quizzes/{classId}/{row.id}/edit">Düzenle</a>{/if}
				<a class={ghost} href="/quizzes/{classId}/{row.id}">Aç <ArrowRight class="size-4" /></a>
			{/snippet}
		</CrudPage>
	{/key}
{:else if !classes.loading}
	<p class="text-zinc-600 dark:text-zinc-400">Henüz bir sınıfa kayıtlı değilsin.</p>
{/if}
