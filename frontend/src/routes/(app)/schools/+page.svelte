<script lang="ts">
	import { ArrowRight, Rocket } from '@lucide/svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import FormDialog from '$lib/components/FormDialog.svelte';
	import { api } from '$lib/api';
	import { ghost } from '$lib/ui';
	import type { Field } from '$lib/options';

	const fields: Field[] = [
		{ name: 'name', label: 'Okul adı', required: true },
		{ name: 'code', label: 'Okul kodu', required: true }
	];
	const startFields: Field[] = [
		...fields,
		{ name: 'campus_name', label: 'İlk kampüs adı' },
		{ name: 'campus_code', label: 'İlk kampüs kodu' }
	];
	let startOpen = $state(false);
	let refresh = $state(0);

	async function start(v: Record<string, unknown>) {
		await api.schools.start({
			school: { name: String(v.name), code: String(v.code) },
			campuses: v.campus_name ? [{ name: String(v.campus_name), code: String(v.campus_code ?? '') }] : []
		});
		refresh++;
	}
</script>

{#key refresh}
	<CrudPage
		title="Okullar"
		subtitle="Okulları ve personellerini yönet."
		columns={[{ key: 'name', label: 'Okul' }, { key: 'code', label: 'Kod' }, { key: 'created_at', label: 'Oluşturulma' }]}
		{fields}
		load={() => api.schools.list()}
		createLabel="Yeni okul"
		create={(v) => api.schools.create(v)}
		update={(r, v) => api.schools.update(String(r.id), v)}
		remove={(r) => api.schools.remove(String(r.id))}
	>
		{#snippet headerExtra()}
			<button class={ghost} onclick={() => (startOpen = true)}><Rocket class="size-4" />Okulu kampüsle başlat</button>
		{/snippet}
		{#snippet extra(row)}
			<a class={ghost} href="/schools/{row.id}">Personel <ArrowRight class="size-4" /></a>
		{/snippet}
	</CrudPage>
{/key}

<FormDialog bind:open={startOpen} title="Okulu kampüsle başlat" submitLabel="Başlat" fields={startFields} onsubmit={start} />
