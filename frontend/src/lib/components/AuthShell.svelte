<script lang="ts">
	import type { Snippet } from 'svelte';
	import { GraduationCap } from '@lucide/svelte';
	import ThemeToggle from './ThemeToggle.svelte';

	let {
		title,
		subtitle,
		children,
		footer
	}: { title: string; subtitle: string; children: Snippet; footer: Snippet } = $props();
</script>

<div class="flex min-h-dvh flex-col bg-zinc-50 dark:bg-zinc-950">
	<header class="flex items-center justify-between px-5 py-4 sm:px-8">
		<div class="flex items-center gap-3 text-lg font-bold">
			<span class="grid size-10 place-items-center rounded-lg bg-brand-600 text-white">
				<GraduationCap class="size-6" />
			</span>
			Kampüs
		</div>
		<ThemeToggle />
	</header>

	<main class="flex flex-1 items-center justify-center px-5 pt-4 pb-14">
		<div
			class="w-full max-w-md space-y-8 rounded-lg border border-t-4 border-zinc-200 border-t-brand-600 shadow-sm
				bg-white p-6 sm:p-10 dark:border-zinc-800 dark:border-t-brand-400 dark:bg-zinc-900"
		>
			<div class="space-y-2">
				<h1 class="text-4xl font-bold tracking-tight">{title}</h1>
				<p class="text-base text-zinc-600 dark:text-zinc-400">{subtitle}</p>
			</div>

			{@render children()}

			<p class="text-center text-base text-zinc-600 dark:text-zinc-400">
				{@render footer()}
			</p>
		</div>
	</main>
</div>
