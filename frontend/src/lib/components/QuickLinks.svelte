<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { navFor } from '$lib/nav';

	// Panelden hızlı erişim: yan menüyle aynı kurallardan (rol + URL koruması) beslenir.
	const links = $derived(navFor(session.user?.role).filter((item) => item.href !== '/dashboard'));
</script>

<section aria-labelledby="quick-links" class="mb-8">
	<h2 id="quick-links" class="mb-3 text-sm font-semibold tracking-wide text-zinc-600 uppercase dark:text-zinc-400">Hızlı erişim</h2>
	<ul class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
		{#each links as item (item.href)}
			<li>
				<a
					href={item.href}
					class="group flex h-full items-start gap-3 rounded-lg border border-zinc-200 bg-white p-4 shadow-sm outline-none transition
						hover:border-brand-300 hover:shadow focus-visible:ring-4 focus-visible:ring-brand-600/30
						dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-brand-700"
				>
					<span class="grid size-10 shrink-0 place-items-center rounded-md bg-brand-50 text-brand-700 transition group-hover:bg-brand-100 dark:bg-brand-950 dark:text-brand-300">
						<item.icon class="size-5" />
					</span>
					<span class="min-w-0">
						<span class="block font-semibold">{item.label}</span>
						<span class="block text-sm text-zinc-600 dark:text-zinc-400">{item.hint}</span>
					</span>
				</a>
			</li>
		{/each}
	</ul>
</section>
