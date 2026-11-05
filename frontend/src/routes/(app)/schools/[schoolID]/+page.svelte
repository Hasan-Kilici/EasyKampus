<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { idMap, lookup, personName } from '$lib/format';
	import { userOptions, STAFF_OPTS, type Field } from '$lib/options';
	import { ghost } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const id = $derived(page.params.schoolID!);
	const school = resource(() => api.schools.get(id) as Promise<Rec>);
	const people = resource(() => (api.users.list() as unknown as Promise<Rec[]>).catch(() => [] as Rec[]));
	const who = $derived(idMap(people.data ?? [], personName));

	const fields: Field[] = [
		{ name: 'user_id', label: 'Kullanıcı', type: 'select', options: userOptions(), required: true },
		{ name: 'role', label: 'Görev', type: 'select', options: STAFF_OPTS, required: true }
	];
</script>

<PageHeader title={String(school.data?.name ?? 'Okul')} subtitle="Okul personeli">
	{#snippet actions()}<a class={ghost} href="/schools">Okullara dön</a>{/snippet}
</PageHeader>

<CrudPage compact title="Personel" createLabel="Personel ata"
	columns={[{ key: 'user_id', label: 'Kullanıcı', format: (r) => lookup(who, r.user_id) }, { key: 'role', label: 'Görev' }]}
	{fields}
	load={() => api.schools.staff(id).list()}
	create={(v) => api.schools.staff(id).add(v as { user_id: string; role: 'director' | 'registrar' })}
	remove={(r) => api.schools.staff(id).remove(String(r.user_id))} />
