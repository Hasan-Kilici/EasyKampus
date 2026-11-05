<script lang="ts">
	import { Sun, Moon } from '@lucide/svelte';

	let dark = $state(false);

	$effect(() => {
		dark = document.documentElement.classList.contains('dark');
	});

	function toggle() {
		dark = !dark;
		document.documentElement.classList.toggle('dark', dark);
		try {
			localStorage.setItem('theme', dark ? 'dark' : 'light');
		} catch {}
	}
</script>

<button
	type="button"
	onclick={toggle}
	aria-label={dark ? 'Açık temaya geç' : 'Koyu temaya geç'}
	class="grid size-11 place-items-center rounded-lg border border-zinc-200 outline-none transition
		hover:bg-zinc-100 focus-visible:ring-4 focus-visible:ring-brand-600/30
		dark:border-zinc-800 dark:hover:bg-zinc-900"
>
	{#if dark}<Sun class="size-5" />{:else}<Moon class="size-5" />{/if}
</button>
