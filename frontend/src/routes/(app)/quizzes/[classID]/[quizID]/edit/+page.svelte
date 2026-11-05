<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ExamBuilder from '$lib/components/ExamBuilder.svelte';
	import Status from '$lib/components/Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { fromDetail } from '$lib/exam';

	const cid = $derived(page.params.classID!);
	const qid = $derived(page.params.quizID!);
	const detail = resource(() => api.class(cid).exam(qid).get());
</script>

<PageHeader title="Sınavı düzenle" subtitle={detail.data?.quiz.name ?? ''} />

<Status res={detail}>
	{#if detail.data}
		{#key detail.data.quiz.id}
			<ExamBuilder
				classId={cid}
				quizId={qid}
				initial={fromDetail(detail.data)}
				readonly={!detail.data.questions_editable}
				lockedReason={`Bu sınava ${detail.data.attempt_count} deneme yapıldı; geçmiş cevaplar bozulmasın diye sorular değiştirilemez.`}
			/>
		{/key}
	{/if}
</Status>
