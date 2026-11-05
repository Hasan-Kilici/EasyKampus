<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Tabs from '$lib/components/Tabs.svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import ReviewButtons from '$lib/components/ReviewButtons.svelte';
	import WorkloadPanel from '$lib/components/WorkloadPanel.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { ghost, card } from '$lib/ui';
	import type { Field } from '$lib/options';
	import type { ContentAnalysis, Rec, TopicRelevance } from '$lib/api/types';

	const id = $derived(page.params.courseID!);
	const course = resource(() => api.courses.get(id) as Promise<Rec>);
	let tab = $state('lessons');
	const m = $derived(session.canManage);

	const lessonFields: Field[] = [
		{ name: 'title', label: 'Başlık', required: true },
		{ name: 'description', label: 'Açıklama', type: 'textarea' },
		{ name: 'markdown_content', label: 'İçerik', type: 'textarea' },
		{ name: 'duration_minutes', label: 'Süre (dk)', type: 'number' },
		{ name: 'position', label: 'Sıra', type: 'number' }
	];
	const topicFields: Field[] = [
		{ name: 'name', label: 'Konu adı', required: true },
		{ name: 'code', label: 'Kod' },
		{ name: 'description', label: 'Bu konu ne anlatıyor? (en az 10 karakter)', type: 'textarea', required: true, placeholder: 'Konunun kapsamını ve öğrenciye ne kazandıracağını yazın. Bölümle ilgi puanı bu metinden hesaplanır.' },
		{ name: 'study_hours', label: 'Konuya ayrılacak süre (saat)', type: 'number' },
		{ name: 'weight', label: 'Ağırlık', type: 'number' },
		{ name: 'active', label: 'Aktif', type: 'checkbox' }
	];

	// Konu–bölüm ilgisi (TF-IDF + kosinüs): konu listesiyle birlikte tek istekte yüklenir.
	const BAND: Record<string, string> = { high: 'Yüksek', medium: 'Orta', low: 'Düşük', none: 'İlgisiz' };
	let analysis = $state<ContentAnalysis | null>(null);
	async function loadTopics(): Promise<Rec[]> {
		const [topics, content] = await Promise.all([
			api.course(id).topics.list(),
			api.course(id).contentAnalysis().catch(() => null)
		]);
		analysis = content;
		const byId = new Map<string, TopicRelevance>((content?.topics ?? []).map((t) => [t.topic_id, t]));
		return (Array.isArray(topics) ? topics : []).map((topic) => {
			const hit = content?.ready && topic.active ? byId.get(String(topic.id)) : undefined;
			return {
				...topic,
				relevance_label: hit ? `${Math.round(hit.score)} · ${BAND[hit.band] ?? hit.band}` : '—',
				matched_label: hit && hit.matched_terms.length ? hit.matched_terms.join(', ') : '—',
				description_short: String(topic.description ?? '').trim() ? String(topic.description).slice(0, 80) : 'Açıklama yok'
			} as Rec;
		});
	}
</script>

<PageHeader title={String(course.data?.name ?? 'Ders')} subtitle={String(course.data?.description ?? '')} />
<Tabs items={[{ id: 'lessons', label: 'İçerik' }, { id: 'topics', label: 'Konular' }, { id: 'workload', label: 'EFOR / AKTS' }]} bind:value={tab} />

{#key `${tab}-${id}`}
	{#if tab === 'lessons'}
		<CrudPage
			compact
			title="Ders içerikleri"
			columns={[{ key: 'position', label: 'Sıra' }, { key: 'title', label: 'Başlık' }, { key: 'duration_minutes', label: 'Süre (dk)' }, { key: 'state', label: 'Durum' }]}
			fields={lessonFields}
			load={() => api.course(id).lessons.list()}
			createLabel="Yeni içerik"
			create={m ? (v) => api.course(id).lessons.create({ ...v, course_id: id }) : undefined}
			update={m ? (r, v) => api.course(id).lessons.update(String(r.id), v) : undefined}
			remove={m ? (r) => api.course(id).lessons.remove(String(r.id)) : undefined}
		>
			{#snippet extra(row, reload)}
				<ReviewButtons current={String(row.state)} onreview={async (a) => { await api.course(id).lessons.review(String(row.id))({ action: a }); await reload(); }} />
				<a class={ghost} href="/courses/{id}/lessons/{row.id}">Oku</a>
			{/snippet}
		</CrudPage>
	{:else if tab === 'workload'}
		{#if course.data}
			<WorkloadPanel courseId={id} course={course.data} canEdit={session.isAdmin} onSaved={() => course.reload()} />
		{/if}
	{:else}
		<div class="{card} mb-4">
			<h2 class="mb-2 text-xl font-bold">Bölüm uyumu</h2>
			{#if !analysis}
				<p class="text-zinc-600 dark:text-zinc-400">Analiz yükleniyor ya da şu an alınamıyor.</p>
			{:else if !analysis.ready}
				<p class="text-zinc-600 dark:text-zinc-400">Bu dersin bölümü girilmemiş. Dersler sayfasında dersi düzenleyip <strong>Bölüm</strong> alanını doldurursanız konular bölümle ilgisine göre puanlanır.</p>
			{:else}
				<p><span class="text-4xl font-bold">{Math.round(analysis.fit)}</span> <span class="font-semibold">/ 100 · {analysis.department}</span></p>
				<p class="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
					Yöntem: TF-IDF + kosinüs benzerliği (konu adı ve açıklaması ↔ bölüm adı, anahtar kelimeler ve bölüm sözlüğü).
					Profil terimleri: {analysis.department_terms.join(', ') || '—'}
				</p>
			{/if}
		</div>
		<CrudPage
			compact
			title="Konular"
			columns={[{ key: 'name', label: 'Konu' }, { key: 'description_short', label: 'Açıklama' }, { key: 'study_hours', label: 'Süre (saat)' }, { key: 'relevance_label', label: 'Bölüm ilgisi' }, { key: 'matched_label', label: 'Eşleşen terimler' }, { key: 'weight', label: 'Ağırlık' }, { key: 'active', label: 'Aktif' }]}
			fields={topicFields}
			load={loadTopics}
			createLabel="Yeni konu"
			create={m ? (v) => api.course(id).topics.create({ ...v, course_id: id }) : undefined}
			update={m ? (r, v) => api.course(id).topics.update(String(r.id), v) : undefined}
			remove={m ? (r) => api.course(id).topics.remove(String(r.id)) : undefined}
		/>
	{/if}
{/key}
