<script lang="ts">
	import { page } from '$app/state';
	import { solid, card } from '$lib/ui';
</script>

<div class="mx-auto mt-24 max-w-md px-4">
	<div class={card} role="alert">
		<h1 class="text-3xl font-bold">{page.status}</h1>
		<p class="mt-2 text-zinc-600 dark:text-zinc-400">{page.error?.message ?? 'Beklenmeyen bir hata oluştu.'}</p>
		<a class="{solid} mt-5" href="/dashboard">Panele dön</a>
	</div>
</div>
