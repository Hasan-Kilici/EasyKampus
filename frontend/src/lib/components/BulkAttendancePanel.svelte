<script lang="ts">
	import { Users } from '@lucide/svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Status from '$lib/components/Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { idMap, lookup, personName } from '$lib/format';
	import { ATTENDANCE_OPTS } from '$lib/options';
	import { solid, card, input } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let { classId, onSaved }: { classId: string; onSaved?: () => void } = $props();

	const roster = resource(() => api.class(classId).enrollments.list());
	const classCourses = resource(() => api.class(classId).courses.list());
	const people = resource(() => (api.users.list() as unknown as Promise<Rec[]>).catch(() => [] as Rec[]));
	const courses = resource(() => api.courses.list().catch(() => [] as Rec[]));

	const who = $derived(idMap(people.data ?? [], personName));
	const what = $derived(idMap(courses.data ?? []));
	const students = $derived.by(() => {
		const byStudent = new Map<string, Rec>();
		for (const row of roster.data ?? []) {
			if (row.status !== 'active') continue;
			const key = String(row.student_id);
			if (!byStudent.has(key)) byStudent.set(key, row);
		}
		return [...byStudent.values()];
	});
	const columns = $derived([
		{ key: 'student_id', label: 'Öğrenci', format: (row: Rec) => lookup(who, row.student_id) },
		{ key: 'status', label: 'Durum' },
		{ key: 'note', label: 'Not' }
	]);

	let selected = $state<string[]>([]);
	let statuses = $state<Record<string, string>>({});
	let notes = $state<Record<string, string>>({});
	let courseId = $state('');
	let date = $state(new Date().toISOString().slice(0, 10));
	let busy = $state(false);
	let err = $state('');
	let notice = $state('');
	let initialized = false;

	$effect(() => {
		const active = students;
		for (const row of active) {
			if (!statuses[String(row.student_id)]) statuses[String(row.student_id)] = 'present';
		}
		if (!initialized && active.length > 0) {
			selected = active.map((row) => String(row.student_id));
			initialized = true;
		}
		if (!courseId) {
			const offered = classCourses.data?.[0]?.course_id;
			if (offered) courseId = offered;
		}
	});

	async function submit() {
		err = '';
		notice = '';
		if (!courseId) {
			err = 'Önce bir ders seç.';
			return;
		}
		if (!date) {
			err = 'Tarih zorunlu.';
			return;
		}
		const rows = students.filter((row) => selected.includes(String(row.student_id)));
		if (rows.length === 0) {
			err = 'En az bir öğrenci seç.';
			return;
		}
		busy = true;
		try {
			await api.class(classId).attendance.record(
				rows.map((row) => ({
					class_id: classId,
					course_id: courseId,
					student_id: String(row.student_id),
					date,
					status: statuses[String(row.student_id)] ?? 'present',
					note: notes[String(row.student_id)] || undefined
				}))
			);
			notice = `${rows.length} öğrenci için yoklama kaydedildi.`;
			onSaved?.();
		} catch (e) {
			err = errorMessage(e);
		} finally {
			busy = false;
		}
	}
</script>

<section class="{card} mb-4">
	<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
		<h2 class="inline-flex items-center gap-2 font-semibold"><Users class="size-5" />Toplu yoklama</h2>
		<span class="text-sm opacity-70">{selected.length} / {students.length} seçili</span>
	</div>

	<div class="mb-3 flex flex-wrap items-end gap-3">
		<label class="text-sm font-semibold">
			Ders
			<select class="{input} mt-1 min-w-52" bind:value={courseId}>
				<option value="">Seç…</option>
				{#each classCourses.data ?? [] as row (row.id)}
					<option value={row.course_id}>{lookup(what, row.course_id)}</option>
				{/each}
			</select>
		</label>
		<label class="text-sm font-semibold">
			Tarih
			<input class="{input} mt-1" type="date" bind:value={date} />
		</label>
		<button class={solid} disabled={busy} onclick={submit}>
			{busy ? 'Kaydediliyorâ€¦' : 'Toplu kaydet'}
		</button>
	</div>

	<Status res={roster} emptyText="Bu sınıfta kayıtlı öğrenci yok.">
		<DataTable
			{columns}
			rows={students as unknown as Rec[]}
			rowKey={(row) => String(row.student_id)}
			selectable
			bind:selected
		>
			{#snippet cell(row, column)}
				{#if column.key === 'status'}
					<select
						class="{input} !h-9 !py-0"
						value={statuses[String(row.student_id)] ?? 'present'}
						onchange={(event) => (statuses[String(row.student_id)] = (event.currentTarget as HTMLSelectElement).value)}
					>
						{#each ATTENDANCE_OPTS as opt}<option value={opt.value}>{opt.label}</option>{/each}
					</select>
				{:else if column.key === 'note'}
					<input
						class="{input} !h-9 !py-0"
						placeholder="Not"
						value={notes[String(row.student_id)] ?? ''}
						oninput={(event) => (notes[String(row.student_id)] = (event.currentTarget as HTMLInputElement).value)}
					/>
				{:else}
					{lookup(who, row.student_id)}
				{/if}
			{/snippet}
		</DataTable>
	</Status>

	{#if err}<p class="mt-2 font-semibold text-red-600" role="alert">{err}</p>{/if}
	{#if notice}<p class="mt-2 font-semibold text-emerald-600" role="status">{notice}</p>{/if}
</section>

