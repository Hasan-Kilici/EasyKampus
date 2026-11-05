<script lang="ts">
	import { ArrowRight } from '@lucide/svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { schoolOptions, type Field } from '$lib/options';
	import { ghost } from '$lib/ui';

	const fields: Field[] = [
		{ name: 'name', label: 'Ders adı', required: true },
		{ name: 'code', label: 'Ders kodu', required: true },
		{ name: 'subject_area', label: 'Alan' },
		{ name: 'department', label: 'Bölüm', placeholder: 'örn. Bilgisayar Programcılığı' },
		{ name: 'department_keywords', label: 'Bölüm için ek anahtar kelimeler (virgülle)', type: 'textarea', placeholder: 'örn. yazılım, donanım, veri tabanı' },
		{ name: 'school_id', label: 'Okul', type: 'select', options: schoolOptions, required: true },
		{ name: 'description', label: 'Açıklama', type: 'textarea' },
		{ name: 'homework_hours', label: 'Ödeve ayrılacak süre (saat)', type: 'number' },
		{ name: 'project_hours', label: 'Projeye ayrılacak süre (saat)', type: 'number' },
		{ name: 'active', label: 'Aktif', type: 'checkbox' }
	];
</script>

<CrudPage
	title="Dersler"
	subtitle="Ders içeriklerine ve konulara buradan ulaş."
	columns={[{ key: 'code', label: 'Kod' }, { key: 'name', label: 'Ders' }, { key: 'subject_area', label: 'Alan' }, { key: 'department', label: 'Bölüm' }, { key: 'active', label: 'Aktif' }]}
	{fields}
	load={() => api.courses.list()}
	createLabel="Yeni ders"
	create={session.isAdmin ? (v) => api.courses.create(v) : undefined}
	update={session.isAdmin ? (r, v) => api.courses.update(String(r.id), v) : undefined}
	remove={session.isAdmin ? (r) => api.courses.remove(String(r.id)) : undefined}
>
	{#snippet extra(row)}
		<a class={ghost} href="/courses/{row.id}">Aç <ArrowRight class="size-4" /></a>
	{/snippet}
</CrudPage>
