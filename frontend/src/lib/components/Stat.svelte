<script lang="ts">
	import { card } from '$lib/ui';
	let { label, value, hint = '' }: { label: string; value: string | number; hint?: string } = $props();
</script>

<div class={card}>
	<p class="text-sm font-medium text-zinc-600 dark:text-zinc-400">{label}</p>
	<p class="mt-1 text-3xl font-semibold tracking-tight">{value}</p>
	{#if hint}<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{hint}</p>{/if}
</div>
