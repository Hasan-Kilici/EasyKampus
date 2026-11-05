<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Status from '$lib/components/Status.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { ghost } from '$lib/ui';

	const id = $derived(page.params.gradeID!);
	const res = resource(() => api.grades.audit(id));
</script>

<PageHeader title="Not değişiklik geçmişi">
	{#snippet actions()}<a class={ghost} href="/grades">Notlara dön</a>{/snippet}
</PageHeader>

<Status {res} empty={!res.data?.length} emptyText="Bu not için kayıtlı değişiklik yok.">
	<DataTable
		columns={[{ key: 'created_at', label: 'Tarih' }, { key: 'action', label: 'İşlem' }, { key: 'before_score', label: 'Önceki puan' }, { key: 'after_score', label: 'Yeni puan' }, { key: 'before_note', label: 'Önceki not' }, { key: 'after_note', label: 'Yeni not' }]}
		rows={res.data ?? []}
	/>
</Status>
