<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Status from '$lib/components/Status.svelte';
	import PerformanceView from '$lib/components/PerformanceView.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import type { StudentPerformance, TeacherPerformance } from '$lib/api/types';

	// 503: Python analiz servisi kapalıysa Status bileşeni net bir mesaj gösterir
	const res = resource<StudentPerformance | TeacherPerformance>(() =>
		session.isStudent ? api.students.me.performance() : api.teachers.mePerformance()
	);
</script>

<PageHeader title="Performans" subtitle="Genel durumun, güçlü ve zayıf yönlerin." />

<Status {res}>
	{#if res.data}<PerformanceView data={res.data} />{/if}
</Status>
