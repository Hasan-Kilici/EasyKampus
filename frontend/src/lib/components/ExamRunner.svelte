<script lang="ts">
	import { untrack } from 'svelte';
	import { Clock } from '@lucide/svelte';
	import ConfirmDialog from './ConfirmDialog.svelte';
	import { api } from '$lib/api';
	import { ApiError, errorMessage } from '$lib/api/client';
	import { card, ghost, solid } from '$lib/ui';
	import { formatClock, remainingMs } from '$lib/exam';
	import type { TakeExam } from '$lib/api/types';

	// Sınav çözme ekranı: süre sunucu saatine göre sayılır, her seçim hemen kaydedilir,
	// süre bitince otomatik teslim edilir. Doğru/yanlış bilgisi sınav sürerken gösterilmez.
	let { exam, attemptId, onfinished }: { exam: TakeExam; attemptId: string; onfinished: () => void } = $props();

	const fetchedAt = Date.now();
	let now = $state(Date.now());
	let answers = $state<Record<string, string>>(untrack(() => ({ ...exam.answers })));
	let saving = $state<Record<string, 'saving' | 'saved' | 'error'>>({});
	let confirmOpen = $state(false);
	let submitting = $state(false);
	let err = $state('');

	const chain = new Map<string, Promise<void>>();
	const total = $derived(exam.questions.length);
	const answered = $derived(exam.questions.filter((q) => answers[q.id]).length);
	const left = $derived(remainingMs(exam.deadline, exam.server_time, fetchedAt, now));
	const urgent = $derived(left !== null && left <= 60_000);

	$effect(() => {
		const timer = setInterval(() => { now = Date.now(); }, 1000);
		return () => clearInterval(timer);
	});

	// Süre bitti: kayıtlı cevaplarla teslim et (sunucu da süre dolunca kendisi tamamlar).
	let autoSubmitted = false;
	$effect(() => {
		if (left === 0 && !autoSubmitted) {
			autoSubmitted = true;
			void submit();
		}
	});

	function pick(questionId: string, optionId: string | null) {
		const previous = answers[questionId];
		if (optionId === null) delete answers[questionId];
		else answers[questionId] = optionId;
		saving[questionId] = 'saving';
		// Aynı soru için istekler sırayla gider; eski bir yanıt yenisinin üzerine yazmaz.
		const run = (chain.get(questionId) ?? Promise.resolve()).then(async () => {
			try {
				await api.attempts.answer(attemptId, { question_id: questionId, selected_option_id: optionId ?? '' });
				saving[questionId] = 'saved';
			} catch (e) {
				if (e instanceof ApiError && (e.status === 403 || e.status === 409)) { onfinished(); return; } // süre dolmuş
				if (previous) answers[questionId] = previous; else delete answers[questionId];
				saving[questionId] = 'error';
				err = `Cevabın kaydedilemedi: ${errorMessage(e)}`;
			}
		});
		chain.set(questionId, run);
	}

	async function submit() {
		if (submitting) return;
		submitting = true; err = '';
		try {
			await Promise.all([...chain.values()]);
			await api.attempts.submit(attemptId);
			onfinished();
		} catch (e) {
			err = errorMessage(e);
			autoSubmitted = false;
		} finally { submitting = false; }
	}
</script>

<div class="sticky top-0 z-10 -mx-4 mb-6 border-b border-zinc-200 bg-white/95 px-4 py-3 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/95">
	<div class="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-xl font-bold">{exam.quiz.name}</h1>
			<p class="text-sm font-semibold text-zinc-600 dark:text-zinc-400">{answered} / {total} soru cevaplandı</p>
		</div>
		<div class="flex items-center gap-3">
			{#if left !== null}
				<p role="timer" aria-label="Kalan süre" class="flex items-center gap-2 rounded-lg border px-3 py-1.5 text-lg font-bold tabular-nums {urgent ? 'border-red-600 text-red-600' : 'border-zinc-950 dark:border-zinc-300'}">
					<Clock class="size-5" />{formatClock(left)}
				</p>
			{/if}
			<button class={solid} disabled={submitting} onclick={() => (confirmOpen = true)}>{submitting ? 'Teslim ediliyor…' : 'Sınavı teslim et'}</button>
		</div>
	</div>
</div>

<div class="mx-auto max-w-3xl">
	{#if exam.quiz.description}<p class="mb-5 text-zinc-600 dark:text-zinc-400">{exam.quiz.description}</p>{/if}
	{#if err}<p class="mb-4 font-semibold text-red-600" role="alert">{err}</p>{/if}

	<ol class="space-y-5">
		{#each exam.questions as q, i (q.id)}
			<li class={card} aria-labelledby={`q-${q.id}`}>
				<p id={`q-${q.id}`} class="mb-1 text-lg font-bold">{i + 1}. {q.question}</p>
				<p class="mb-4 text-sm text-zinc-600 dark:text-zinc-400">{q.points} puan</p>
				<div role="radiogroup" aria-labelledby={`q-${q.id}`} class="space-y-2">
					{#each q.options as o (o.id)}
						{@const on = answers[q.id] === o.id}
						<button
							role="radio"
							aria-checked={on}
							onclick={() => pick(q.id, o.id)}
							class="flex min-h-12 w-full items-center gap-3 rounded-lg border px-4 py-2 text-left font-semibold outline-none transition focus-visible:ring-4 focus-visible:ring-brand-600/30
								{on ? 'border-brand-600 bg-brand-50 text-brand-900 dark:border-brand-400 dark:bg-brand-950 dark:text-brand-50' : 'border-zinc-200 hover:bg-zinc-100 dark:border-zinc-800 dark:hover:bg-zinc-800'}"
						>
							<span class="grid size-5 shrink-0 place-items-center rounded-full border border-current">
								{#if on}<span class="size-2.5 rounded-full bg-current"></span>{/if}
							</span>
							{o.content}
						</button>
					{/each}
				</div>
				<div class="mt-3 flex items-center justify-between text-sm">
					<button class={ghost} type="button" disabled={!answers[q.id]} onclick={() => pick(q.id, null)}>Temizle</button>
					<span aria-live="polite" class="font-semibold {saving[q.id] === 'error' ? 'text-red-600' : 'text-zinc-600 dark:text-zinc-400'}">
						{#if saving[q.id] === 'saving'}Kaydediliyor…{:else if saving[q.id] === 'saved'}Kaydedildi ✓{:else if saving[q.id] === 'error'}Kaydedilemedi{/if}
					</span>
				</div>
			</li>
		{/each}
	</ol>

	<div class="mt-6 flex justify-end">
		<button class={solid} disabled={submitting} onclick={() => (confirmOpen = true)}>Sınavı teslim et</button>
	</div>
</div>

<ConfirmDialog
	bind:open={confirmOpen}
	title="Sınavı teslim et"
	text={total - answered > 0 ? `${total - answered} soru boş kaldı. Teslim ettikten sonra cevapları değiştiremezsin. Teslim edilsin mi?` : 'Teslim ettikten sonra cevapları değiştiremezsin. Teslim edilsin mi?'}
	onconfirm={submit}
/>
