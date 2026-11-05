<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ghost, card } from '$lib/ui';

	let {
		res,
		empty = false,
		emptyText = 'Henüz kayıt yok.',
		children
	}: {
		res: { loading: boolean; error: string; data: unknown; reload: () => unknown };
		empty?: boolean;
		emptyText?: string;
		children: Snippet;
	} = $props();
</script>

{#if res.loading && !res.data}
	<div class="space-y-3" aria-busy="true" aria-label="Yükleniyor">
		{#each [0, 1, 2] as i (i)}<div class="h-14 animate-pulse rounded-lg bg-zinc-200 dark:bg-zinc-800"></div>{/each}
	</div>
{:else if res.error}
	<div class={card} role="alert">
		<p class="font-semibold text-red-600 dark:text-red-400">{res.error}</p>
		<button class="{ghost} mt-4" onclick={() => res.reload()}>Tekrar dene</button>
	</div>
{:else if empty}
	<div class="{card} text-center text-zinc-600 dark:text-zinc-400">{emptyText}</div>
{:else}
	{@render children()}
{/if}
