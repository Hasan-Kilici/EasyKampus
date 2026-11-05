<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AppShell from '$lib/components/AppShell.svelte';
	import { session } from '$lib/session.svelte';
	import { setUnauthorizedHandler } from '$lib/api/client';
	import { HOME, canAccess } from '$lib/access';
	import { card, solid } from '$lib/ui';

	let { children } = $props();

	$effect(() => {
		setUnauthorizedHandler(() => goto('/login'));
		session.load().then(() => {
			if (!session.user && !session.error) goto('/login');
		});
		return () => setUnauthorizedHandler(null);
	});

	// URL ile yetkisiz sayfaya gitmeyi engelle: içerik hiç çizilmez, istek atılmaz, panele yönlendirilir.
	const allowed = $derived(canAccess(page.url.pathname, session.user?.role));
	$effect(() => {
		if (session.user && !allowed) goto(HOME, { replaceState: true });
	});
</script>

{#if session.user}
	<AppShell>
		{#if allowed}{@render children()}{:else}<p class="text-zinc-600 dark:text-zinc-400" role="status">Bu sayfaya erişim yetkin yok. Panele yönlendiriliyorsun…</p>{/if}
	</AppShell>
{:else if session.error}
	<div class="mx-auto mt-24 max-w-md px-4">
		<div class={card} role="alert">
			<p class="font-semibold text-red-600 dark:text-red-400">{session.error}</p>
			<p class="mt-2 text-sm text-zinc-600 dark:text-zinc-400">Oturumun silinmedi. Birkaç saniye sonra tekrar dene.</p>
			<button class="{solid} mt-4" onclick={() => session.load()}>Tekrar dene</button>
		</div>
	</div>
{/if}
