<script lang="ts">
	import type { Snippet } from 'svelte';
	let { title, subtitle, actions }: { title: string; subtitle?: string; actions?: Snippet } = $props();
</script>

<svelte:head><title>{title} · Kampüs</title></svelte:head>

<div class="mb-6 flex flex-wrap items-end justify-between gap-4">
	<div class="space-y-1">
		<h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
		{#if subtitle}<p class="text-sm text-zinc-600 dark:text-zinc-400">{subtitle}</p>{/if}
	</div>
	{#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
</div>
