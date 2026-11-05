<script lang="ts">
	import { Field } from '@ark-ui/svelte/field';
	import { Checkbox } from '@ark-ui/svelte/checkbox';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { Eye, EyeOff, Check } from '@lucide/svelte';
	import { input, label, error, button } from '$lib/ui';
	import { loginSchema, fieldErrors } from '$lib/schemas';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { ApiError, token } from '$lib/api/client';
	import { session } from '$lib/session.svelte';

	let email = $state('');
	let password = $state('');
	let remember = $state(false);
	let showPassword = $state(false);
	let loading = $state(false);
	let errors = $state<{ email?: string; password?: string; form?: string }>({});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		const result = loginSchema.safeParse({ email, password, remember });
		if (!result.success) {
			errors = fieldErrors(result.error);
			return;
		}
		errors = {};

		loading = true;
		try {
			const res = await api.auth.login({ email: result.data.email, password: result.data.password });
			token.set(res.token, remember);
			await session.load();
			await goto('/dashboard');
		} catch (e) {
			errors = { form: e instanceof ApiError ? e.message : 'Sunucuya ulaşılamadı. Tekrar dene.' };
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Giriş yap · Kampüs</title>
</svelte:head>

<AuthShell title="Tekrar hoş geldin" subtitle="Derslerine ve ödevlerine devam etmek için giriş yap.">
	<form onsubmit={handleSubmit} novalidate class="space-y-5">
		<Field.Root invalid={!!errors.email}>
			<Field.Label class={label}>E-posta</Field.Label>
			<Field.Input
				type="email"
				autocomplete="email"
				bind:value={email}
				placeholder="ad@okul.edu.tr"
				class={input}
			/>
			<Field.ErrorText class={error}>{errors.email}</Field.ErrorText>
		</Field.Root>

		<Field.Root invalid={!!errors.password}>
			<div class="mb-2 flex items-center justify-between">
				<Field.Label class="{label} !mb-0">Şifre</Field.Label>
				
			</div>

			<div class="relative">
				<Field.Input
					type={showPassword ? 'text' : 'password'}
					autocomplete="current-password"
					bind:value={password}
					placeholder="Şifren"
					class="{input} pr-14"
				/>
				<button
					type="button"
					onclick={() => (showPassword = !showPassword)}
					aria-label={showPassword ? 'Şifreyi gizle' : 'Şifreyi göster'}
					aria-pressed={showPassword}
					class="absolute inset-y-0 right-0 grid w-12 place-items-center rounded-r-xl text-sm font-bold
						outline-none focus-visible:ring-4 focus-visible:ring-brand-600/30"
				>
					{#if showPassword}
						<EyeOff class="size-5" />
					{:else}
						<Eye class="size-5" />
					{/if}
				</button>
			</div>
			<Field.ErrorText class={error}>{errors.password}</Field.ErrorText>
		</Field.Root>

		<Checkbox.Root bind:checked={remember} class="flex min-h-11 cursor-pointer items-center gap-3">
			<Checkbox.Control
				class="grid size-6 place-items-center rounded-md border border-zinc-300 transition
					data-[state=checked]:border-brand-600 data-[state=checked]:bg-brand-600
					data-[focus-visible]:ring-4 data-[focus-visible]:ring-brand-600/30
					dark:border-zinc-600"
			>
				<Checkbox.Indicator class="text-white">
					<Check class="size-4" strokeWidth={3.5} />
				</Checkbox.Indicator>
			</Checkbox.Control>
			<Checkbox.Label class="text-base">Beni hatırla</Checkbox.Label>
			<Checkbox.HiddenInput />
		</Checkbox.Root>

		{#if errors.form}<p class={error} role="alert">{errors.form}</p>{/if}

		<button type="submit" class={button} disabled={loading}>
			{loading ? 'Giriş yapılıyor…' : 'Giriş yap'}
		</button>
	</form>

	{#snippet footer()}
		Hesabın yok mu?
		<a
			href="/register"
			class="rounded font-bold underline underline-offset-4 outline-none focus-visible:ring-4 focus-visible:ring-brand-600/30"
		>
			Kayıt ol
		</a>
	{/snippet}
</AuthShell>
