<script lang="ts">
	import { Play, Trash2, Upload, Check } from '@lucide/svelte';
	import Status from '$lib/components/Status.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { errorMessage, uploadToUrl } from '$lib/api/client';
	import { session } from '$lib/session.svelte';
	import { solid, ghost, card } from '$lib/ui';
	import type { Video } from '$lib/api/types';

	let { courseId, lessonId, canManage = false }: { courseId: string; lessonId: string; canManage?: boolean } = $props();

	const videos = resource(() => api.course(courseId).videos(lessonId).list());

	let playing = $state<{ id: string; url: string } | null>(null);
	let playErr = $state('');
	let loadingPlay = $state('');

	let file = $state<File | null>(null);
	let title = $state('');
	let progress = $state(0);
	let uploading = $state(false);
	let uploadErr = $state('');
	let notice = $state('');

	let deleting = $state<Video | null>(null);
	let confirmOpen = $state(false);
	let deleteErr = $state('');

	$effect(() => {
		if (!confirmOpen) deleting = null;
	});

	function askDelete(video: Video) {
		deleting = video;
		confirmOpen = true;
	}

	async function play(video: Video) {
		playErr = '';
		loadingPlay = video.id;
		try {
			const result = await api.course(courseId).videos(lessonId).play(video.id);
			playing = { id: video.id, url: result.url };
			if (session.isStudent) {
				await api.course(courseId).videos(lessonId).complete(video.id).catch(() => undefined);
			}
		} catch (e) {
			playErr = errorMessage(e);
		} finally {
			loadingPlay = '';
		}
	}

	function pick(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		file = input.files?.[0] ?? null;
		if (file && !title.trim()) title = file.name;
	}

	async function upload() {
		if (!file) {
			uploadErr = 'Önce bir video dosyası seç.';
			return;
		}
		uploading = true;
		uploadErr = '';
		notice = '';
		progress = 0;
		try {
			const contentType = file.type || 'video/mp4';
			const presign = await api.course(courseId).videos(lessonId).presign({
				filename: file.name,
				content_type: contentType
			});
			if (!presign.configured || !presign.upload_url) {
				throw new Error(presign.message ?? 'Sunucuda R2 yapılandırılmamış.');
			}
			await uploadToUrl(presign.upload_url, file, contentType, (p) => (progress = p));
			const url = presign.public_url || `r2://${presign.key}`;
			await api.course(courseId).videos(lessonId).create({
				title: title.trim() || file.name,
				url,
				position: (videos.data ?? []).length
			});
			file = null;
			title = '';
			notice = 'Video yüklendi.';
			await videos.reload();
		} catch (e) {
			uploadErr = errorMessage(e);
		} finally {
			uploading = false;
		}
	}

	async function confirmDelete() {
		if (!deleting) return;
		deleteErr = '';
		try {
			await api.course(courseId).videos(lessonId).remove(deleting.id);
			if (playing?.id === deleting.id) playing = null;
			confirmOpen = false;
			await videos.reload();
		} catch (e) {
			deleteErr = errorMessage(e);
		}
	}
</script>

<section class="mt-6 max-w-3xl">
	<h2 class="mb-3 text-lg font-semibold">Videolar</h2>

	<Status res={videos} emptyText="Bu içerik için henüz video eklenmemiş.">
		{#if videos.data && videos.data.length > 0}
			<ul class="space-y-3">
				{#each videos.data as video (video.id)}
					<li class="{card} p-3">
						<div class="flex flex-wrap items-center justify-between gap-2">
							<div>
								<p class="font-medium">{video.title}</p>
								{#if video.duration > 0}
									<p class="text-sm opacity-70">{Math.floor(video.duration / 60)} dk {video.duration % 60} sn</p>
								{/if}
							</div>
							<div class="flex items-center gap-2">
								<button class={solid} disabled={loadingPlay === video.id} onclick={() => play(video)}>
									<Play class="size-4" />
									{loadingPlay === video.id ? 'Açılıyor…' : 'Oynat'}
								</button>
								{#if canManage}
									<button class="{ghost} !px-3" aria-label="Videoyu sil" onclick={() => askDelete(video)}>
										<Trash2 class="size-4" />
									</button>
								{/if}
							</div>
						</div>
						{#if playing?.id === video.id}
							<!-- svelte-ignore a11y_media_has_caption -->
							<video class="mt-3 w-full rounded-lg bg-black" src={playing.url} controls autoplay playsinline></video>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
		{#if playErr}<p class="mt-2 font-semibold text-red-600" role="alert">{playErr}</p>{/if}
	</Status>

	{#if canManage}
		<div class="{card} mt-4 p-4">
			<h3 class="mb-3 font-semibold">Video yükle</h3>
			<div class="flex flex-col gap-3">
				<input class="text-sm" type="file" accept="video/mp4,video/webm,video/quicktime,video/ogg" onchange={pick} />
				<input
					class="rounded-md border border-black/10 bg-transparent px-3 py-2 text-sm dark:border-white/15"
					placeholder="Başlık"
					bind:value={title}
				/>
				{#if uploading}
					<div class="h-2 w-full overflow-hidden rounded bg-black/10 dark:bg-white/10">
						<div class="h-full bg-emerald-500 transition-all" style={`width:${progress}%`}></div>
					</div>
					<p class="text-sm opacity-70">Yükleniyor… %{progress}</p>
				{/if}
				<div class="flex items-center gap-3">
					<button class={solid} disabled={uploading || !file} onclick={upload}>
						<Upload class="size-4" />
						{uploading ? 'Yükleniyor…' : 'Yükle'}
					</button>
					{#if notice}
						<span class="inline-flex items-center gap-1 font-semibold text-emerald-600" role="status">
							<Check class="size-4" />{notice}
						</span>
					{/if}
				</div>
				{#if uploadErr}<p class="font-semibold text-red-600" role="alert">{uploadErr}</p>{/if}
			</div>
		</div>
	{/if}
</section>

<ConfirmDialog
	bind:open={confirmOpen}
	title="Videoyu sil"
	text={deleting ? `"${deleting.title}" videosu silinsin mi?` : ''}
	onconfirm={confirmDelete}
/>
{#if deleteErr}<p class="mt-2 font-semibold text-red-600" role="alert">{deleteErr}</p>{/if}
