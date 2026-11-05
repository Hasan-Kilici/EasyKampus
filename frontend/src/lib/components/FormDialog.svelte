<script lang="ts">
	import { untrack } from 'svelte';
	import { input, label, error as errCls, solid, ghost, textarea } from '$lib/ui';
	import { errorMessage } from '$lib/api/client';
	import { toMinutes, fmtTime } from '$lib/format';
	import type { Rec } from '$lib/api/types';
	import type { Field, Opt } from '$lib/options';

	let {
		open = $bindable(false),
		title,
		fields,
		initial = {},
		submitLabel = 'Kaydet',
		isCreate = true,
		onsubmit
	}: {
		open: boolean;
		title: string;
		fields: Field[];
		initial?: Rec;
		submitLabel?: string;
		isCreate?: boolean;
		onsubmit: (values: Rec) => Promise<unknown>;
	} = $props();

	const required = (f: Field) =>
		Boolean(f.required || (isCreate ? f.requiredOnCreate : f.requiredOnUpdate));

	let dlg: HTMLDialogElement;
	let values = $state<Record<string, any>>({});
	let opts = $state<Record<string, Opt[]>>({});
	let err = $state('');
	let busy = $state(false);

	const pad = (n: number) => String(n).padStart(2, '0');

	const getTime = (v: unknown, type: 'datetime' | 'date') => {
		if (!v) return '';

		const d = new Date(String(v));

		if (Number.isNaN(d.getTime())) return '';

		const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

		if (type === 'date') {
			return date;
		}

		return `${date}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
	};

	function init() {
		values = {};

		for (const f of fields) {
			if (f.type === 'checkbox') {
				values[f.name] = initial[f.name] ?? false;
			} else if (f.type === 'datetime' || f.type === 'date') {
				values[f.name] = getTime(initial[f.name], f.type);
			} else if (f.type === 'time') {
				values[f.name] = fmtTime(Number(initial[f.name] ?? 0));
			} else {
				values[f.name] = initial[f.name] ?? f.default ?? '';
			}
		}
	}

	async function loadOpts() {
		for (const f of fields) {
			if (f.type !== 'select' || !f.options) continue;

			try {
				opts[f.name] =
					typeof f.options === 'function'
						? await f.options(values)
						: f.options;
			} catch {
				opts[f.name] = [];
			}
		}
	}

	function payload(): Rec {
		const out: Rec = {};

		for (const f of fields) {
			const v = values[f.name];

			if (f.type === 'checkbox') {
				out[f.name] = v;
			} else if (v === '' || v == null) {
				continue;
			} else if (
				f.type === 'number' ||
				(f.type === 'select' && f.numeric)
			) {
				out[f.name] = Number(v);
			} else if (f.type === 'datetime') {
				out[f.name] = new Date(v).toISOString();
			} else if (f.type === 'date') {
				out[f.name] = v;
			} else if (f.type === 'time') {
				out[f.name] = toMinutes(v);
			} else {
				out[f.name] = v;
			}
		}

		return out;
	}

	$effect(() => {
		if (open) {
			untrack(() => {
				init();
				err = '';
				loadOpts();
			});

			if (!dlg.open) {
				dlg.showModal();
			}
		} else if (dlg?.open) {
			dlg.close();
		}
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();

		busy = true;
		err = '';

		try {
			await onsubmit(payload());
			open = false;
		} catch (x) {
			err = errorMessage(x);
		} finally {
			busy = false;
		}
	}
</script>

<dialog
	bind:this={dlg}
	onclose={() => (open = false)}
	class="m-auto max-h-[90dvh] w-[min(32rem,92vw)] overflow-y-auto rounded-lg border border-zinc-200 bg-white p-6 text-zinc-950 backdrop:bg-black/50 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-50"
>
	<h2 class="mb-5 text-2xl font-bold">{title}</h2>

	<form onsubmit={submit} class="space-y-4">
		{#each fields as f (f.name)}
			{#if f.type === 'checkbox'}
				<label class="flex min-h-11 items-center gap-3 font-semibold">
					<input
						type="checkbox"
						bind:checked={values[f.name]}
						class="size-5 accent-brand-600"
					/>
					{f.label}
				</label>
			{:else}
				<div>
					<label class={label} for="f-{f.name}">
						{f.label}
					</label>

					{#if f.type === 'textarea'}
						<textarea
							id="f-{f.name}"
							rows="5"
							class={textarea}
							bind:value={values[f.name]}
							required={required(f)}
							placeholder={f.placeholder}
						></textarea>

					{:else if f.type === 'select'}
						<select
							id="f-{f.name}"
							class={input}
							bind:value={values[f.name]}
							required={required(f)}
						>
							<option value="">Seçin</option>

							{#each opts[f.name] ?? [] as o (o.value)}
								<option value={o.value}>
									{o.label}
								</option>
							{/each}
						</select>

					{:else}
						<input
							id="f-{f.name}"
							type={f.type === 'datetime'
								? 'datetime-local'
								: (f.type ?? 'text')}
							class={input}
							value={values[f.name]}
							oninput={(e) =>
								(values[f.name] = e.currentTarget.value)}
							required={required(f)}
							placeholder={f.placeholder}
						/>
					{/if}
				</div>
			{/if}
		{/each}

		{#if err}
			<p class={errCls} role="alert">{err}</p>
		{/if}

		<div class="flex justify-end gap-2 pt-2">
			<button
				type="button"
				class={ghost}
				onclick={() => (open = false)}
			>
				Vazgeç
			</button>

			<button
				type="submit"
				class={solid}
				disabled={busy}
			>
				{busy ? 'Kaydediliyor…' : submitLabel}
			</button>
		</div>
	</form>
</dialog>