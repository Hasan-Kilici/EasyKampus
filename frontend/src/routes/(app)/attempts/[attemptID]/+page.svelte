<script lang="ts">
	import { page } from '$app/state';
	import ExamRunner from '$lib/components/ExamRunner.svelte';
	import ExamResult from '$lib/components/ExamResult.svelte';
	import Status from '$lib/components/Status.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { ApiError } from '$lib/api/client';

	const aid = $derived(page.params.attemptID!);
	const back = $derived.by(() => {
		const cid = page.url.searchParams.get('class');
		const qid = page.url.searchParams.get('quiz');
		return cid && qid ? `/quizzes/${cid}/${qid}` : '/quizzes';
	});

	// Deneme sürüyorsa sınav ekranı, bittiyse (409) sonuç ekranı gösterilir.
	const view = resource(async () => {
		try {
			return { exam: await api.attempts.exam(aid) };
		} catch (e) {
			if (e instanceof ApiError && e.status === 409) return { exam: null };
			throw e;
		}
	});
</script>

<Status res={view}>
	{#if view.data?.exam}
		{#key aid}
			<ExamRunner exam={view.data.exam} attemptId={aid} onfinished={() => view.reload()} />
		{/key}
	{:else if view.data}
		<ExamResult attemptId={aid} backHref={back} />
	{/if}
</Status>
