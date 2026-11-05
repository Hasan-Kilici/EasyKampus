<script lang="ts">
	let { items, value = $bindable() }: { items: { id: string; label: string }[]; value: string } = $props();
</script>

<div role="tablist" class="mb-6 flex gap-1 overflow-x-auto border-b border-zinc-200 dark:border-zinc-800">
	{#each items as t (t.id)}
		<button
			role="tab"
			aria-selected={value === t.id}
			onclick={() => (value = t.id)}
			class="-mb-px h-10 shrink-0 border-b-2 px-4 text-sm font-medium outline-none focus-visible:ring-4 focus-visible:ring-brand-600/30
				{value === t.id
				? 'border-brand-600 text-brand-700 dark:border-brand-400 dark:text-brand-300'
				: 'border-transparent text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100'}"
		>
			{t.label}
		</button>
	{/each}
</div>
