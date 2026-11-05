<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { ghost } from '$lib/ui';
	import type { Field } from '$lib/options';
	import type { Rec } from '$lib/api/types';

	const cid = $derived(page.params.classID!);
	const qid = $derived(page.params.quizID!);
	const qn = $derived(page.params.questionID!);
	const question = resource(() => api.class(cid).questions(qid).get(qn) as Promise<Rec>);
	const o = () => api.class(cid).questions(qid).options(qn);

	const fields: Field[] = [
		{ name: 'content', label: 'Şık metni', required: true },
		{ name: 'position', label: 'Sıra', type: 'number' },
		{ name: 'answer', label: 'Doğru cevap', type: 'checkbox' }
	];
</script>

<PageHeader title="Şıklar" subtitle={String(question.data?.question ?? '')}>
	{#snippet actions()}<a class={ghost} href="/quizzes/{cid}/{qid}">Sınava dön</a>{/snippet}
</PageHeader>

<CrudPage compact title="Cevap şıkları" createLabel="Yeni şık"
	columns={[{ key: 'position', label: 'Sıra' }, { key: 'content', label: 'Şık' }, { key: 'answer', label: 'Doğru' }]}
	{fields}
	load={() => o().list()}
	create={(v) => o().create({ ...v, question_id: qn })}
	update={(r, v) => o().update(String(r.id), v)}
	remove={(r) => o().remove(String(r.id))} />
