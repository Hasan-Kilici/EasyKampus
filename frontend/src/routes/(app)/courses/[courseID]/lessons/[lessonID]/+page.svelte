<script lang="ts">
	import { page } from '$app/state';
	import { Check } from '@lucide/svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Status from '$lib/components/Status.svelte';
	import VideoSection from '$lib/components/VideoSection.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { solid, ghost, card } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const cid = $derived(page.params.courseID!);
	const lid = $derived(page.params.lessonID!);
	// Öğrenci için "view" çağrısı görüntülenmeyi kaydeder ve içeriği döner
	const res = resource(() => (session.isStudent ? api.course(cid).lessons.view(lid) : api.course(cid).lessons.get(lid)) as Promise<Rec>);
	let done = $state(false);
	let err = $state('');

	async function complete() {
		err = '';
		try { await api.course(cid).lessons.complete(lid); done = true; } catch (e) { err = errorMessage(e); }
	}
</script>

<PageHeader title={String(res.data?.title ?? 'Ders içeriği')} subtitle={res.data?.duration_minutes ? `${res.data.duration_minutes} dakika` : ''} />

<Status {res}>
	<!-- Markdown HTML'e çevrilmeden düz metin olarak gösterilir (XSS riski yok) -->
	<article class="{card} max-w-3xl whitespace-pre-wrap text-lg leading-8">{res.data?.markdown_content ?? 'Bu içerik boş.'}</article>
	<div class="mt-6 flex flex-wrap items-center gap-3">
		<a class={ghost} href="/courses/{cid}">Derse dön</a>
		{#if session.isStudent}
			<button class={solid} disabled={done} onclick={complete}><Check class="size-5" />{done ? 'Tamamlandı' : 'Tamamlandı olarak işaretle'}</button>
		{/if}
		{#if err}<span class="font-semibold text-red-600" role="alert">{err}</span>{/if}
	</div>
	<VideoSection courseId={cid} lessonId={lid} canManage={session.canManage} />
</Status>
