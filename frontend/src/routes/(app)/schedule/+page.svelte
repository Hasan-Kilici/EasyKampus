<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Status from '$lib/components/Status.svelte';
	import ScheduleTable from '$lib/components/ScheduleTable.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { idMap, lookup } from '$lib/format';
	import { input, label } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const none = () => [] as Rec[];
	const res = resource(async () => {
		const [rows, courses, classes] = await Promise.all([
			api.schedules.list(), api.courses.list().catch(none), api.classes.list().catch(none)
		]);
		return { rows, courses: idMap(courses), classes: idMap(classes) };
	});

	// Sunucu zaten kullanıcının görebildiği programı döner; burada yalnızca sınıfa göre süzülür.
	let classFilter = $state('');
	const classIds = $derived([...new Set((res.data?.rows ?? []).map((r) => String(r.class_id)))]);
	const rows = $derived((res.data?.rows ?? []).filter((r) => !classFilter || String(r.class_id) === classFilter));
	const showClass = $derived(!classFilter && classIds.length > 1);

	const subtitle = $derived(
		session.isStudent
			? 'Sınıfının haftalık ders tablosu.'
			: 'Haftalık ders tablosu. Programı değiştirmek için ilgili sınıfın “Program” sekmesini kullan.'
	);
</script>

<PageHeader title="Ders programı" {subtitle} />

{#if classIds.length > 1}
	<div class="mb-4 max-w-xs">
		<label class={label} for="class-filter">Sınıf</label>
		<select id="class-filter" class={input} bind:value={classFilter}>
			<option value="">Tüm sınıflar</option>
			{#each classIds as id (id)}<option value={id}>{lookup(res.data!.classes, id)}</option>{/each}
		</select>
	</div>
{/if}

<Status {res} empty={!res.data?.rows.length} emptyText="Programda ders yok.">
	{#if res.data}<ScheduleTable {rows} courses={res.data.courses} classes={res.data.classes} {showClass} />{/if}
</Status>
