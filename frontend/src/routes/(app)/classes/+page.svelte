<script lang="ts">
	import { ArrowRight } from '@lucide/svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { schoolOptions, campusOptions, type Field } from '$lib/options';
	import { ghost } from '$lib/ui';

	const fields: Field[] = [
		{ name: 'name', label: 'Sınıf adı', required: true },
		{ name: 'no', label: 'Şube (tek harf)', required: true },
		{ name: 'level', label: 'Seviye', required: true },
		{ name: 'school_id', label: 'Okul', type: 'select', options: schoolOptions, required: true },
		{ name: 'campus_id', label: 'Kampüs', type: 'select', options: campusOptions, required: true }
	];
</script>

<CrudPage
	title="Sınıflar"
	subtitle="Öğrenci, öğretmen, ders programı, yoklama ve notlar sınıf içinden yönetilir."
	columns={[{ key: 'name', label: 'Sınıf' }, { key: 'no', label: 'Şube' }, { key: 'level', label: 'Seviye' }]}
	{fields}
	load={() => api.classes.list()}
	createLabel="Yeni sınıf"
	create={session.isAdmin ? (v) => api.classes.create(v) : undefined}
	update={session.isAdmin ? (r, v) => api.classes.update(String(r.id), v) : undefined}
	remove={session.isAdmin ? (r) => api.classes.remove(String(r.id)) : undefined}
>
	{#snippet extra(row)}
		<a class={ghost} href="/classes/{row.id}">Yönet <ArrowRight class="size-4" /></a>
	{/snippet}
</CrudPage>
