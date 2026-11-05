<script lang="ts">
	import { ghost } from '$lib/ui';

	let {
		open = $bindable(false),
		title,
		text = '',
		onconfirm
	}: { open: boolean; title: string; text?: string; onconfirm: () => Promise<void> } = $props();

	let dlg: HTMLDialogElement;
	let busy = $state(false);

	$effect(() => {
		if (open) { if (!dlg.open) dlg.showModal(); }
		else if (dlg?.open) dlg.close();
	});

	async function go() {
		busy = true;
		await onconfirm();
		busy = false;
		open = false;
	}
</script>

<dialog
	bind:this={dlg}
	onclose={() => (open = false)}
	class="m-auto w-[min(26rem,92vw)] rounded-lg border border-zinc-200 bg-white p-6 text-zinc-950 backdrop:bg-black/50 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-50"
>
	<h2 class="text-xl font-bold">{title}</h2>
	{#if text}<p class="mt-2 text-zinc-600 dark:text-zinc-400">{text}</p>{/if}
	<div class="mt-6 flex justify-end gap-2">
		<button type="button" class={ghost} onclick={() => (open = false)}>Vazgeç</button>
		<button type="button" class="{ghost} !border-red-600 !text-red-600" disabled={busy} onclick={go}>
			{busy ? 'Siliniyor…' : 'Sil'}
		</button>
	</div>
</dialog>
