<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Play } from '@lucide/svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Tabs from '$lib/components/Tabs.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Status from '$lib/components/Status.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { fmt, idMap, lookup, personName } from '$lib/format';
	import { card, error as errCls, ghost, notice, solid } from '$lib/ui';
	import type { Rec, ReviewAction } from '$lib/api/types';

	const cid = $derived(page.params.classID!);
	const qid = $derived(page.params.quizID!);
	const staff = $derived(session.canManage);

	// Öğrenci görünümü: sınav bilgisi + kendi denemeleri
	const quiz = resource(() => (session.isStudent ? (api.class(cid).quizzes.get(qid) as Promise<Rec>) : Promise.resolve(null)));
	const attempts = resource(async () => (session.isStudent ? (await api.attempts.list()).filter((a) => String(a.quiz_id) === qid) : []));
	// Personel görünümü: cevap anahtarlı ayrıntı
	const detail = resource(() => (session.isStudent ? Promise.resolve(null) : api.class(cid).exam(qid).get()));

	let tab = $state('preview');
	let busy = $state(false);
	let err = $state('');

	const active = $derived(attempts.data?.find((a) => a.status === 'in_progress'));
	const used = $derived(attempts.data?.filter((a) => a.status === 'completed').length ?? 0);
	const maxAttempts = $derived(Number(quiz.data?.max_attempts ?? 1));

	async function start() {
		busy = true; err = '';
		try {
			const a = await api.attempts.start(qid);
			await goto(`/attempts/${a.id}?class=${cid}&quiz=${qid}`);
		} catch (e) { err = errorMessage(e); busy = false; }
	}

	async function review(action: ReviewAction) {
		busy = true; err = '';
		try { await api.class(cid).exam(qid).review({ action }); await detail.reload(); }
		catch (e) { err = errorMessage(e); } finally { busy = false; }
	}

	// Sonuçlar sekmesi: açıldığında yüklenir
	const results = resource(async () => (tab === 'results' && staff ? api.class(cid).exam(qid).results() : null));
	const names = resource(async () => (staff ? idMap(await api.users.list().catch(() => [] as Rec[]), personName) : new Map<string, string>()));

	const when = (v: unknown) => (v && !String(v).startsWith('0001') ? new Date(String(v)).toLocaleString('tr-TR') : '—');
</script>

{#if session.isStudent}
	<PageHeader title={String(quiz.data?.name ?? 'Sınav')} subtitle={String(quiz.data?.description ?? '')} />
	<div class="{card} mb-6 flex flex-wrap items-center justify-between gap-4">
		<div class="space-y-1">
			<p class="font-bold">{Number(quiz.data?.duration_minutes) ? `${quiz.data?.duration_minutes} dakika` : 'Süresiz'} · {used} / {maxAttempts} deneme kullanıldı</p>
			<p class="text-sm text-zinc-600 dark:text-zinc-400">Açılış: {when(quiz.data?.open_at)} · Kapanış: {when(quiz.data?.close_at)}</p>
			{#if err}<p class={errCls} role="alert">{err}</p>{/if}
		</div>
		{#if active}
			<a class={solid} href="/attempts/{active.id}?class={cid}&quiz={qid}"><Play class="size-5" />Sınava devam et</a>
		{:else if used >= maxAttempts}
			<p class="font-semibold">Deneme hakkın bitti.</p>
		{:else}
			<button class={solid} disabled={busy} onclick={start}><Play class="size-5" />Sınava başla</button>
		{/if}
	</div>

	<h2 class="mb-3 text-xl font-bold">Denemelerin</h2>
	<Status res={attempts} empty={!attempts.data?.length} emptyText="Bu sınava henüz girmedin.">
		<DataTable
			columns={[{ key: 'started_at', label: 'Başlangıç' }, { key: 'status', label: 'Durum' }, { key: 'score', label: 'Puan' }, { key: 'max_score', label: 'Tam puan' }]}
			rows={(attempts.data ?? []).map((a) => ({ ...a, score: a.status === 'completed' ? a.score : '—', max_score: a.status === 'completed' ? a.max_score : '—' }))}
		>
			{#snippet actions(row)}
				<a class={ghost} href="/attempts/{row.id}?class={cid}&quiz={qid}">{row.status === 'in_progress' ? 'Devam et' : 'Sonuç'}</a>
			{/snippet}
		</DataTable>
	</Status>
{:else}
	<PageHeader title={detail.data?.quiz.name ?? 'Sınav'} subtitle={detail.data?.quiz.description ?? ''}>
		{#snippet actions()}
			{#if detail.data}
				{@const state = String(detail.data.quiz.state)}
				<span class="rounded-full border border-current px-3 py-1 text-sm font-bold">{fmt('state', state)}</span>
				<a class={ghost} href="/quizzes/{cid}/{qid}/edit">Düzenle</a>
				{#if staff && (state === 'draft' || state === 'rejected')}
					<button class={ghost} disabled={busy} onclick={() => review('submit')}>Onaya gönder</button>
				{/if}
				{#if session.isAdmin && (state === 'draft' || state === 'rejected' || state === 'pending')}
					<button class={solid} disabled={busy} onclick={() => review('approve')}>{state === 'pending' ? 'Onayla' : 'Yayınla'}</button>
				{/if}
				{#if session.isAdmin && state === 'pending'}
					<button class={ghost} disabled={busy} onclick={() => review('reject')}>Reddet</button>
				{/if}
			{/if}
		{/snippet}
	</PageHeader>
	{#if err}<p class="{errCls} mb-4" role="alert">{err}</p>{/if}

	<Status res={detail}>
		{#if detail.data}
			{@const d = detail.data}
			<div class="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<Stat label="Soru" value={d.questions.length} />
				<Stat label="Toplam puan" value={d.quiz.points} />
				<Stat label="Süre" value={d.quiz.duration_minutes ? `${d.quiz.duration_minutes} dk` : 'Süresiz'} hint={`${d.quiz.max_attempts} deneme hakkı`} />
				<Stat label="Çözen" value={d.attempt_count} hint="deneme sayısı" />
			</div>

			{#if d.issues.length && d.quiz.state !== 'approved'}
				<div class="{notice} mb-6" role="status">
					<p class="mb-2 font-bold">Yayın için eksikler</p>
					<ul class="list-disc space-y-1 pl-5">{#each d.issues as issue (issue.path + issue.message)}<li>{issue.message}</li>{/each}</ul>
					<a class="{ghost} mt-3" href="/quizzes/{cid}/{qid}/edit">Düzenleyerek tamamla</a>
				</div>
			{/if}

			<Tabs items={[{ id: 'preview', label: 'Sorular' }, { id: 'results', label: 'Sonuçlar' }]} bind:value={tab} />
			{#if tab === 'preview'}
				<ol class="space-y-4">
					{#each d.questions as q, i (q.id)}
						<li class={card}>
							<p class="mb-3 text-lg font-bold">{i + 1}. {q.question} <span class="text-sm font-normal text-zinc-600 dark:text-zinc-400">({q.points} puan)</span></p>
							<ul class="space-y-1.5">
								{#each q.options as o (o.id)}
									<li class="flex items-center gap-2 rounded-lg border px-3 py-1.5 font-semibold {o.answer ? 'border-green-600 bg-green-50 dark:bg-green-950' : 'border-zinc-200 dark:border-zinc-800'}">
										{o.content}{#if o.answer}<span class="ml-auto text-sm text-green-700 dark:text-green-400">Doğru cevap</span>{/if}
									</li>
								{/each}
							</ul>
						</li>
					{/each}
				</ol>
			{:else}
				<Status res={results} empty={!results.data}>
					{#if results.data}
						{@const r = results.data}
						<div class="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
							<Stat label="Katılımcı" value={r.participants} />
							<Stat label="Ortalama" value={`%${r.average_percentage}`} hint="en iyi denemelere göre" />
							<Stat label="En yüksek" value={`%${r.highest_percentage}`} />
							<Stat label="En düşük" value={`%${r.lowest_percentage}`} />
						</div>
						<h2 class="mb-3 text-xl font-bold">Öğrenciler</h2>
						<DataTable
							columns={[{ key: 'student', label: 'Öğrenci' }, { key: 'attempts', label: 'Deneme' }, { key: 'best', label: 'En iyi' }, { key: 'last', label: 'Son' }]}
							rows={r.students.map((s) => ({ id: s.student_id, student: lookup(names.data ?? new Map(), s.student_id), attempts: s.attempts, best: `${s.best_score} / ${s.max_score} (%${s.best_percentage})`, last: `${s.last_score} / ${s.max_score}` }))}
						/>
						<h2 class="mb-3 mt-8 text-xl font-bold">Soru bazında başarı</h2>
						<DataTable
							columns={[{ key: 'n', label: '#' }, { key: 'question', label: 'Soru' }, { key: 'answered', label: 'Cevaplayan' }, { key: 'correct', label: 'Doğru' }, { key: 'rate', label: 'Başarı' }]}
							rows={r.questions.map((x, i) => ({ id: x.question_id, n: i + 1, question: x.question, answered: x.answered, correct: x.correct, rate: x.answered ? `%${x.rate}` : '—' }))}
						/>
					{/if}
				</Status>
			{/if}
		{/if}
	</Status>
{/if}
