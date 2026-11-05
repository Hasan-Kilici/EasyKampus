<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { GraduationCap, Menu, X, LogOut } from '@lucide/svelte';
	import ThemeToggle from './ThemeToggle.svelte';
	import { session } from '$lib/session.svelte';
	import { navFor } from '$lib/nav';

	let { children }: { children: Snippet } = $props();
	let open = $state(false);

	const roleLabel = { superadmin: 'Süper yönetici', admin: 'Yönetici', teacher: 'Öğretmen', student: 'Öğrenci' };
	const items = $derived(navFor(session.user?.role));
	const initials = $derived(((session.user?.name[0] ?? '') + (session.user?.surname[0] ?? '')).toUpperCase());

	async function logout() {
		await session.logout();
		await goto('/login');
	}
</script>

<div class="min-h-dvh bg-zinc-50 dark:bg-zinc-950">
	<!-- Üst bar: Moodle / üniversite paneli düzeni -->
	<header
		class="sticky top-0 z-40 flex h-16 items-center gap-3 border-b border-zinc-200 bg-white px-4
			dark:border-zinc-800 dark:bg-zinc-900"
	>
		<button
			type="button"
			class="grid size-11 place-items-center rounded-lg outline-none hover:bg-zinc-100 focus-visible:ring-4
				focus-visible:ring-brand-600/30 lg:hidden dark:hover:bg-zinc-800"
			aria-label={open ? 'Menüyü kapat' : 'Menüyü aç'}
			onclick={() => (open = !open)}
		>
			{#if open}<X class="size-5" />{:else}<Menu class="size-5" />{/if}
		</button>

		<a href="/dashboard" class="flex items-center gap-3 text-lg font-bold">
			<span class="grid size-9 place-items-center rounded-lg bg-brand-600 text-white">
				<GraduationCap class="size-5" />
			</span>
			Kampüs
		</a>

		<div class="ml-auto flex items-center gap-3">
			<div class="hidden text-right leading-tight sm:block">
				<p class="text-sm font-bold">{session.user?.name} {session.user?.surname}</p>
				<p class="text-xs text-zinc-600 dark:text-zinc-400">{session.user ? roleLabel[session.user.role] : ''}</p>
			</div>
			<span class="grid size-11 place-items-center rounded-full bg-brand-600 text-sm font-bold text-white">
				{initials}
			</span>
			<ThemeToggle />
			<button
				type="button"
				onclick={logout}
				aria-label="Çıkış yap"
				class="grid size-11 place-items-center rounded-lg border border-zinc-200 outline-none transition
					hover:bg-zinc-100 focus-visible:ring-4 focus-visible:ring-brand-600/30
					dark:border-zinc-800 dark:hover:bg-zinc-800"
			>
				<LogOut class="size-5" />
			</button>
		</div>
	</header>

	<div class="lg:grid lg:grid-cols-[16rem_1fr]">
		<!-- Sol menü: masaüstünde sabit, mobilde çekmece -->
		<aside
			class="fixed inset-y-16 left-0 z-30 w-64 overflow-y-auto border-r border-zinc-200 bg-white p-3 transition-transform
				lg:sticky lg:top-16 lg:h-[calc(100dvh-4rem)] lg:translate-x-0 dark:border-zinc-800 dark:bg-zinc-900
				{open ? 'translate-x-0' : '-translate-x-full'}"
		>
			<nav class="space-y-1" aria-label="Ana menü">
				{#each items as item}
					{@const active = page.url.pathname.startsWith(item.href)}
					<a
						href={item.href}
						onclick={() => (open = false)}
						aria-current={active ? 'page' : undefined}
						class="flex h-10 items-center gap-3 rounded-lg border-l-2 px-3 text-sm font-medium outline-none transition
							focus-visible:ring-4 focus-visible:ring-brand-600/30
							{active
							? 'border-brand-600 bg-brand-50 text-brand-800 dark:bg-brand-950/50 dark:text-brand-200'
							: 'border-transparent text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800'}"
					>
						<item.icon class="size-5 shrink-0" />
						{item.label}
					</a>
				{/each}
			</nav>
		</aside>

		<main class="mx-auto w-full max-w-6xl p-4 sm:p-8">{@render children()}</main>
	</div>
</div>
