<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ExamBuilder from '$lib/components/ExamBuilder.svelte';
	import { card } from '$lib/ui';

	const classId = $derived(page.url.searchParams.get('class') ?? '');
</script>

<PageHeader title="Yeni sınav" subtitle="Bilgileri, soruları ve seçenekleri bir arada girin; taslak olarak kaydedin, hazır olunca yayına gönderin." />

{#if classId}
	{#key classId}<ExamBuilder {classId} />{/key}
{:else}
	<div class={card}>Önce <a class="font-semibold underline" href="/quizzes">Sınavlar</a> sayfasından bir sınıf seçip “Yeni sınav”a basın.</div>
{/if}
