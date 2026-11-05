<script lang="ts">
	import { UserPlus } from '@lucide/svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Status from '$lib/components/Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { personName } from '$lib/format';
	import { currentTerm } from '$lib/options';
	import { solid, card, input } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	let { classId, onSaved }: { classId: string; onSaved?: () => void } = $props();

	const roster = resource(() => api.class(classId).enrollments.list());
	const users = resource(() => (api.users.list() as unknown as Promise<Rec[]>).catch(() => [] as Rec[]));

	const enrolled = $derived(
		new Set((roster.data ?? []).filter((row) => row.status === 'active').map((row) => row.student_id))
	);
	const candidates = $derived(
		(users.data ?? []).filter((user) => user.role === 'student' && !enrolled.has(String(user.id)))
	);
	const columns = [
		{ key: 'name', label: 'Öğrenci', format: (row: Rec) => personName(row) },
		{ key: 'email', label: 'E-posta' }
	];

	let selected = $state<string[]>([]);
	let term = $state(currentTerm());
	let note = $state('');
	let busy = $state(false);
	let err = $state('');
	let notice = $state('');

	async function submit() {
		err = '';
		notice = '';
		if (selected.length === 0) {
			err = 'En az bir öğrenci seç.';
			return;
		}
		busy = true;
		try {
			await api.class(classId).enrollments.bulk(
				selected.map((studentId) => ({
					student_id: studentId,
					term: term.trim() || undefined,
					note: note.trim() || undefined
				}))
			);
			notice = `${selected.length} öğrenci kaydedildi.`;
			selected = [];
			await roster.reload();
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
		<h2 class="inline-flex items-center gap-2 font-semibold"><UserPlus class="size-5" />Toplu öğrenci kaydı</h2>
		<span class="text-sm opacity-70">{selected.length} seçili</span>
	</div>

	<p class="mb-3 text-sm text-zinc-600 dark:text-zinc-400">
		Öğrenci sınıfa kaydedilince, sınıfa atanmış tüm derslere otomatik olarak kayıtlı olur; sonradan eklenen dersler de kendiliğinden görünür.
	</p>
	<div class="mb-3 flex flex-wrap items-end gap-3">
		<label class="text-sm font-semibold">
			Dönem
			<input class="{input} mt-1" placeholder="2025-2026 Güz" bind:value={term} />
		</label>
		<label class="text-sm font-semibold">
			Not
			<input class="{input} mt-1" bind:value={note} />
		</label>
		<button class={solid} disabled={busy} onclick={submit}>
			{busy ? 'Kaydediliyor…' : 'Toplu kaydet'}
		</button>
	</div>

	<Status res={users} emptyText="Kayıt edilebilecek öğrenci bulunamadı. Kullanıcı listesi için yönetici yetkisi gerekir.">
		{#if candidates.length === 0}
			<p class="text-sm opacity-70">Bu sınıfa kaydedilebilecek yeni öğrenci yok.</p>
		{:else}
			<DataTable
				{columns}
				rows={candidates as unknown as Rec[]}
				rowKey={(row) => String(row.id)}
				selectable
				bind:selected
			/>
		{/if}
	</Status>

	{#if err}<p class="mt-2 font-semibold text-red-600" role="alert">{err}</p>{/if}
	{#if notice}<p class="mt-2 font-semibold text-emerald-600" role="status">{notice}</p>{/if}
</section>
