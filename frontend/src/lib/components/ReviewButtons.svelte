<script lang="ts">
	import { solid, ghost } from '$lib/ui';
	import { session } from '$lib/session.svelte';
	import { errorMessage } from '$lib/api/client';

	type Action = 'submit' | 'approve' | 'reject';
	let { current, onreview }: { current: string; onreview: (a: Action) => Promise<unknown> } = $props();
	let busy = $state(false);
	let err = $state('');

	async function go(a: Action) {
		busy = true;
		err = '';
		try { await onreview(a); } catch (e) { err = errorMessage(e); } finally { busy = false; }
	}
</script>

{#if session.canManage && (current === 'draft' || current === 'rejected')}
	<button class={ghost} disabled={busy} onclick={() => go('submit')}>Onaya gönder</button>
{:else if session.isAdmin && current === 'pending'}
	<button class={solid} disabled={busy} onclick={() => go('approve')}>Onayla</button>
	<button class={ghost} disabled={busy} onclick={() => go('reject')}>Reddet</button>
{/if}
{#if err}<span class="text-sm font-semibold text-red-600" role="alert">{err}</span>{/if}
