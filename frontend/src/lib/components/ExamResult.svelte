<script lang="ts">
	import { Check, X } from '@lucide/svelte';
	import PageHeader from './PageHeader.svelte';
	import Status from './Status.svelte';
	import Stat from './Stat.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { card, notice } from '$lib/ui';

	let { attemptId, backHref = '' }: { attemptId: string; backHref?: string } = $props();
	const result = resource(() => api.attempts.result(attemptId));

	const minutes = (seconds: number) => (seconds >= 60 ? `${Math.floor(seconds / 60)} dk ${seconds % 60} sn` : `${seconds} sn`);
</script>

<Status res={result}>
	{#if result.data}
		{@const r = result.data}
		<PageHeader title={r.quiz_name || 'Sınav sonucu'} subtitle="Sınav tamamlandı." />
		<div class="mb-6 grid gap-4 sm:grid-cols-3">
			<Stat label="Puan" value={`${r.score} / ${r.max_score}`} />
			<Stat label="Başarı" value={`%${Math.round(r.percentage)}`} />
			<Stat label="Süre" value={minutes(r.duration_seconds)} />
		</div>

		{#if !r.answers_revealed}
			<div class="{notice} mb-6" role="status">
				<p class="font-bold">Doğru cevaplar henüz gösterilmiyor.</p>
				<p class="text-zinc-600 dark:text-zinc-400">Sınav kapandığında ya da deneme hakların bittiğinde doğru cevapları ve soru bazında sonucunu görebilirsin.</p>
			</div>
		{/if}

		<ol class="space-y-4">
			{#each r.questions as q, i (q.question_id)}
				<li class={card}>
					<p class="mb-3 flex items-start justify-between gap-3 text-lg font-bold">
						<span>{i + 1}. {q.question}</span>
						{#if q.correct !== undefined}
							<span class="flex shrink-0 items-center gap-1 text-base {q.correct ? 'text-green-700 dark:text-green-400' : 'text-red-600'}">
								{#if q.correct}<Check class="size-5" />Doğru{:else}<X class="size-5" />{q.answered ? 'Yanlış' : 'Boş'}{/if}
								· {q.earned_points ?? 0}/{q.points}
							</span>
						{/if}
					</p>
					<ul class="space-y-2">
						{#each q.options as o (o.id)}
							<li class="flex items-center gap-3 rounded-lg border px-4 py-2 font-semibold
								{o.correct ? 'border-green-600 bg-green-50 dark:bg-green-950' : o.selected && r.answers_revealed ? 'border-red-500 bg-red-50 dark:bg-red-950' : o.selected ? 'border-brand-600 dark:border-brand-400' : 'border-zinc-200 dark:border-zinc-800'}">
								<span class="grid size-5 shrink-0 place-items-center rounded-full border border-current">
									{#if o.selected}<span class="size-2.5 rounded-full bg-current"></span>{/if}
								</span>
								<span class="flex-1">{o.content}</span>
								{#if o.selected}<span class="text-sm">Senin cevabın</span>{/if}
								{#if o.correct}<span class="text-sm text-green-700 dark:text-green-400">Doğru cevap</span>{/if}
							</li>
						{/each}
					</ul>
				</li>
			{/each}
		</ol>
		{#if backHref}<p class="mt-6"><a class="font-semibold underline" href={backHref}>Sınava dön</a></p>{/if}
	{/if}
</Status>
