<script lang="ts">
	import ScheduleTable from './ScheduleTable.svelte';
	import Status from './Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';

	// Tek bir sınıfın (şubenin) haftalık tablosu.
	let { classId, courses }: { classId: string; courses: Map<string, string> } = $props();
	const rows = resource(() => api.class(classId).schedules.list());
</script>

<Status res={rows} empty={!rows.data?.length} emptyText="Bu sınıfın programı henüz boş.">
	<ScheduleTable rows={rows.data ?? []} {courses} />
</Status>
