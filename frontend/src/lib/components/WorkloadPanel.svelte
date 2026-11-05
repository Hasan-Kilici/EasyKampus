<script lang="ts">
	import { untrack } from 'svelte';
	import Stat from './Stat.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { card, input, label, solid, error as errCls } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	// EFOR: dersin iş yükü = aktif konu saatleri + ödev + proje; AKTS = toplam saat / (bir AKTS'nin saat karşılığı).
	let { courseId, course, canEdit = false, onSaved }: { courseId: string; course: Rec; canEdit?: boolean; onSaved?: () => void } = $props();

	let perEcts = $state(30);
	const workload = resource(() => api.course(courseId).workload(perEcts));

	// Düzenlenebilir alanlar yalnızca ilk değerden başlar; kullanıcı yazdıkça prop'tan bağımsızdır.
	let homework = $state(untrack(() => Number(course.homework_hours ?? 0)));
	let project = $state(untrack(() => Number(course.project_hours ?? 0)));
	let saving = $state(false);
	let message = $state('');
	let failure = $state('');

	async function save() {
		saving = true;
		message = '';
		failure = '';
		try {
			await api.courses.update(courseId, {
				school_id: course.school_id,
				code: course.code,
				name: course.name,
				description: course.description ?? '',
				subject_area: course.subject_area ?? '',
				active: course.active,
				department: course.department ?? '',
				department_keywords: course.department_keywords ?? '',
				homework_hours: Number(homework) || 0,
				project_hours: Number(project) || 0
			});
			message = 'Kaydedildi.';
			await workload.reload();
			onSaved?.();
		} catch (e) {
			failure = errorMessage(e);
		} finally {
			saving = false;
		}
	}
</script>

<div class="space-y-6">
	{#if workload.error}
		<p class={errCls} role="alert">{workload.error}</p>
	{:else if workload.data}
		{@const w = workload.data}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
			<Stat label="Konular" value={w.topic_hours} hint="saat" />
			<Stat label="Ödev" value={w.homework_hours} hint="saat" />
			<Stat label="Proje" value={w.project_hours} hint="saat" />
			<Stat label="Toplam iş yükü" value={w.total_hours} hint="saat" />
			<Stat label="AKTS" value={w.ects} hint={`ham: ${w.ects_exact} · 1 AKTS = ${w.hours_per_ects} saat`} />
		</div>

		<div class={card}>
			<h2 class="mb-3 text-xl font-bold">Konu bazında süre</h2>
			{#if w.topics.length}
				<ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
					{#each w.topics as t (t.topic_id)}
						<li class="flex items-center justify-between py-2"><span>{t.name}</span><span class="font-bold">{t.hours} saat</span></li>
					{/each}
				</ul>
			{:else}
				<p class="text-zinc-600 dark:text-zinc-400">Henüz aktif konu yok. Süreleri “Konular” sekmesinde konu eklerken girersiniz.</p>
			{/if}
		</div>
	{/if}

	<div class={card}>
		<h2 class="mb-3 text-xl font-bold">Hesap ayarları</h2>
		<div class="grid gap-4 sm:grid-cols-3">
			<div>
				<label class={label} for="perEcts">1 AKTS kaç saat?</label>
				<select id="perEcts" class={input} bind:value={perEcts}>
					<option value={25}>25 saat</option>
					<option value={27.5}>27,5 saat</option>
					<option value={30}>30 saat</option>
				</select>
			</div>
			<div>
				<label class={label} for="homework">Ödeve ayrılacak süre (saat)</label>
				<input id="homework" class={input} type="number" min="0" step="0.5" bind:value={homework} disabled={!canEdit} />
			</div>
			<div>
				<label class={label} for="project">Projeye ayrılacak süre (saat)</label>
				<input id="project" class={input} type="number" min="0" step="0.5" bind:value={project} disabled={!canEdit} />
			</div>
		</div>
		{#if canEdit}
			<div class="mt-4 flex items-center gap-3">
				<button class={solid} onclick={save} disabled={saving}>{saving ? 'Kaydediliyor…' : 'Süreleri kaydet'}</button>
				{#if message}<span class="font-semibold" role="status">{message}</span>{/if}
				{#if failure}<span class={errCls} role="alert">{failure}</span>{/if}
			</div>
		{/if}
	</div>
</div>
