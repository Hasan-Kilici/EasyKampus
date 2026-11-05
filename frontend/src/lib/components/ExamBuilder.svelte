<script lang="ts">
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { ArrowDown, ArrowUp, Copy, Plus, Trash2 } from '@lucide/svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { classCourseOptions } from '$lib/options';
	import { card, error as errCls, ghost, input, label, notice, solid, textarea } from '$lib/ui';
	import {
		MAX_OPTIONS, checkDraft, fromDetail, moveItem, newDraft, newOption, newQuestion, toInput, totalPoints,
		type DraftQuestion, type ExamDraft
	} from '$lib/exam';
	import type { ExamDetail } from '$lib/api/types';

	// Sınav oluşturucu: bilgiler + sorular + seçenekler tek formda, tek işlemde kaydedilir.
	let {
		classId,
		quizId = undefined,
		initial = undefined,
		readonly = false,
		lockedReason = ''
	}: { classId: string; quizId?: string; initial?: ExamDraft; readonly?: boolean; lockedReason?: string } = $props();

	let draft = $state<ExamDraft>(untrack(() => initial ?? newDraft()));
	let currentId = $state<string | undefined>(untrack(() => quizId));
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');

	const courses = resource(() => classCourseOptions(classId)());
	const topics = resource(async () => {
		if (!draft.course_id) return [];
		const list = await api.course(draft.course_id).topics.list();
		return (Array.isArray(list) ? list : []).filter((t) => t.active !== false).map((t) => ({ value: String(t.id), label: String(t.name) }));
	});
	$effect(() => {
		if (!draft.course_id && courses.data?.length && !currentId) draft.course_id = courses.data[0].value;
	});

	const issues = $derived(checkDraft(draft));
	const points = $derived(totalPoints(draft));
	const forQuestion = (index: number) => issues.filter((i) => i.path.startsWith(`questions[${index}]`));
	const general = $derived(issues.filter((i) => !i.path.startsWith('questions[')));

	const addQuestion = () => { draft.questions = [...draft.questions, newQuestion()]; };
	const removeQuestion = (i: number) => { draft.questions = draft.questions.filter((_, index) => index !== i); };
	const moveQuestion = (i: number, by: number) => { draft.questions = moveItem(draft.questions, i, i + by); };
	const duplicateQuestion = (i: number) => {
		const source = draft.questions[i];
		const copy: DraftQuestion = { ...newQuestion(), topic_id: source.topic_id, question: source.question, points: source.points, options: source.options.map((o) => newOption(o.content, o.answer)) };
		draft.questions = [...draft.questions.slice(0, i + 1), copy, ...draft.questions.slice(i + 1)];
	};
	const addOption = (q: DraftQuestion) => { if (q.options.length < MAX_OPTIONS) q.options = [...q.options, newOption()]; };
	const removeOption = (q: DraftQuestion, i: number) => { q.options = q.options.filter((_, index) => index !== i); };
	const markCorrect = (q: DraftQuestion, i: number) => { q.options.forEach((o, index) => { o.answer = index === i; }); };

	async function save(): Promise<ExamDetail | null> {
		busy = true; failure = ''; message = '';
		try {
			const body = toInput(draft);
			const detail = currentId ? await api.class(classId).exam(currentId).replace(body) : await api.class(classId).exams.create(body);
			currentId = detail.quiz.id;
			draft = fromDetail(detail);
			message = 'Taslak kaydedildi.';
			return detail;
		} catch (e) { failure = errorMessage(e); return null; } finally { busy = false; }
	}

	async function saveDraft() {
		const wasNew = !currentId;
		const detail = await save();
		if (detail && wasNew) await goto(`/quizzes/${classId}/${detail.quiz.id}/edit`, { replaceState: true });
	}

	async function publish() {
		const detail = await save();
		if (!detail) return;
		busy = true;
		try {
			await api.class(classId).exam(detail.quiz.id).review({ action: session.isAdmin ? 'approve' : 'submit' });
			await goto(`/quizzes/${classId}/${detail.quiz.id}`);
		} catch (e) { failure = errorMessage(e); } finally { busy = false; }
	}
</script>

<div class="space-y-6 pb-28">
	{#if readonly}
		<div class={notice} role="status">
			<p class="font-bold">Bu sınav düzenlenemez.</p>
			<p class="text-zinc-600 dark:text-zinc-400">{lockedReason || 'Sınav çözülmeye başlandığı için sorular değiştirilemez.'}</p>
		</div>
	{/if}

	<section class={card} aria-labelledby="exam-info">
		<h2 id="exam-info" class="mb-4 text-xl font-bold">Sınav bilgileri</h2>
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="sm:col-span-2">
				<label class={label} for="exam-name">Sınav adı</label>
				<input id="exam-name" class={input} bind:value={draft.name} disabled={readonly} />
			</div>
			<div>
				<label class={label} for="exam-course">Ders</label>
				<select id="exam-course" class={input} bind:value={draft.course_id} disabled={readonly || !!currentId}>
					{#each courses.data ?? [] as c (c.value)}<option value={c.value}>{c.label}</option>{/each}
				</select>
				{#if courses.data && !courses.data.length}<p class={errCls}>Bu sınıfa atanmış ders yok.</p>{/if}
			</div>
			<div>
				<label class={label} for="exam-attempts">Deneme hakkı</label>
				<input id="exam-attempts" class={input} type="number" min="1" max="20" bind:value={draft.max_attempts} disabled={readonly} />
			</div>
			<div>
				<label class={label} for="exam-duration">Süre (dakika, 0 = süresiz)</label>
				<input id="exam-duration" class={input} type="number" min="0" max="600" bind:value={draft.duration_minutes} disabled={readonly} />
			</div>
			<div></div>
			<div>
				<label class={label} for="exam-open">Açılış</label>
				<input id="exam-open" class={input} type="datetime-local" bind:value={draft.open_at} disabled={readonly} />
			</div>
			<div>
				<label class={label} for="exam-close">Kapanış</label>
				<input id="exam-close" class={input} type="datetime-local" bind:value={draft.close_at} disabled={readonly} />
			</div>
			<div class="sm:col-span-2">
				<label class={label} for="exam-description">Açıklama</label>
				<textarea id="exam-description" class={textarea} rows="2" bind:value={draft.description} disabled={readonly}></textarea>
			</div>
		</div>
		{#each general as issue (issue.path)}<p class={errCls}>{issue.message}</p>{/each}
	</section>

	<section aria-labelledby="exam-questions" class="space-y-4">
		<div class="flex items-end justify-between">
			<h2 id="exam-questions" class="text-xl font-bold">Sorular ({draft.questions.length})</h2>
			<p class="font-semibold text-zinc-600 dark:text-zinc-400">Toplam {points} puan</p>
		</div>

		{#each draft.questions as q, qi (q.key)}
			{@const problems = forQuestion(qi)}
			<article class={card} aria-label={`Soru ${qi + 1}`}>
				<div class="mb-3 flex items-center justify-between gap-2">
					<h3 class="text-lg font-bold">Soru {qi + 1}</h3>
					{#if !readonly}
						<div class="flex gap-1">
							<button class={ghost} type="button" title="Yukarı taşı" aria-label="Yukarı taşı" disabled={qi === 0} onclick={() => moveQuestion(qi, -1)}><ArrowUp class="size-4" /></button>
							<button class={ghost} type="button" title="Aşağı taşı" aria-label="Aşağı taşı" disabled={qi === draft.questions.length - 1} onclick={() => moveQuestion(qi, 1)}><ArrowDown class="size-4" /></button>
							<button class={ghost} type="button" title="Çoğalt" aria-label="Soruyu çoğalt" onclick={() => duplicateQuestion(qi)}><Copy class="size-4" /></button>
							<button class={ghost} type="button" title="Sil" aria-label="Soruyu sil" onclick={() => removeQuestion(qi)}><Trash2 class="size-4" /></button>
						</div>
					{/if}
				</div>

				<label class={label} for={`q-${q.key}`}>Soru metni</label>
				<textarea id={`q-${q.key}`} class={textarea} rows="2" bind:value={q.question} disabled={readonly}></textarea>

				<div class="mt-3 grid gap-4 sm:grid-cols-2">
					<div>
						<label class={label} for={`p-${q.key}`}>Puan</label>
						<input id={`p-${q.key}`} class={input} type="number" min="0" max="100" step="0.5" bind:value={q.points} disabled={readonly} />
					</div>
					<div>
						<label class={label} for={`t-${q.key}`}>Konu (isteğe bağlı)</label>
						<select id={`t-${q.key}`} class={input} bind:value={q.topic_id} disabled={readonly}>
							<option value="">—</option>
							{#each topics.data ?? [] as t (t.value)}<option value={t.value}>{t.label}</option>{/each}
						</select>
					</div>
				</div>

				<fieldset class="mt-4">
					<legend class={label}>Seçenekler — doğru olanı işaretleyin</legend>
					<div class="space-y-2">
						{#each q.options as o, oi (o.key)}
							<div class="flex items-center gap-3">
								<input type="radio" class="size-5 shrink-0 accent-zinc-950" name={`correct-${q.key}`} aria-label={`${oi + 1}. seçenek doğru cevap`} checked={o.answer} disabled={readonly} onchange={() => markCorrect(q, oi)} />
								<input class={input} aria-label={`${oi + 1}. seçenek metni`} bind:value={o.content} disabled={readonly} />
								{#if !readonly}
									<button class={ghost} type="button" aria-label="Seçeneği sil" disabled={q.options.length <= 2} onclick={() => removeOption(q, oi)}><Trash2 class="size-4" /></button>
								{/if}
							</div>
						{/each}
					</div>
					{#if !readonly && q.options.length < MAX_OPTIONS}
						<button class="{ghost} mt-3" type="button" onclick={() => addOption(q)}><Plus class="size-4" />Seçenek ekle</button>
					{/if}
				</fieldset>

				{#each problems as issue (issue.path)}<p class={errCls}>{issue.message}</p>{/each}
			</article>
		{/each}

		{#if !readonly}
			<button class={ghost} type="button" onclick={addQuestion}><Plus class="size-4" />Soru ekle</button>
		{/if}
	</section>
</div>

{#if !readonly}
	<div class="fixed inset-x-0 bottom-0 z-20 border-t-2 border-zinc-200 bg-white/95 p-3 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/95">
		<div class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
			<p class="text-sm font-semibold" role="status">
				{#if failure}<span class="text-red-600 dark:text-red-400" role="alert">{failure}</span>
				{:else if message}{message}
				{:else if issues.length}Yayına hazır değil: {issues.length} eksik var.
				{:else}Yayına hazır.{/if}
			</p>
			<div class="flex gap-2">
				<button class={ghost} type="button" disabled={busy} onclick={saveDraft}>Taslağı kaydet</button>
				<button class={solid} type="button" disabled={busy || issues.length > 0} onclick={publish}>
					{session.isAdmin ? 'Kaydet ve yayınla' : 'Kaydet ve onaya gönder'}
				</button>
			</div>
		</div>
	</div>
{/if}
