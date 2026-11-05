<script lang="ts">
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { idMap, lookup } from '$lib/format';
	import { schoolOptions, type Field } from '$lib/options';
	import type { Rec } from '$lib/api/types';

	const schools = resource(() => api.schools.list().catch(() => [] as Rec[]));
	const names = $derived(idMap(schools.data ?? []));
	const fields: Field[] = [
		{ name: 'name', label: 'Kampüs adı', required: true },
		{ name: 'code', label: 'Kod', required: true },
		{ name: 'school_id', label: 'Okul', type: 'select', options: schoolOptions, required: true }
	];
</script>

<CrudPage
	title="Kampüsler"
	columns={[{ key: 'name', label: 'Kampüs' }, { key: 'code', label: 'Kod' }, { key: 'school_id', label: 'Okul', format: (r) => lookup(names, r.school_id) }]}
	{fields}
	load={() => api.campuses.list()}
	createLabel="Yeni kampüs"
	create={(v) => api.campuses.create(v)}
	update={(r, v) => api.campuses.update(String(r.id), v)}
	remove={(r) => api.campuses.remove(String(r.id))}
/>
