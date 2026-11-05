import { errorMessage } from '$lib/api/client';

/** Yükleme / hata / yeniden yükleme durumunu yöneten reaktif yardımcı. fn içindeki $state değişince otomatik yeniler. */
export function resource<T>(fn: () => Promise<T>) {
	let data = $state<T | null>(null);
	let loading = $state(true);
	let error = $state('');

	let run = 0;

	async function load() {
		const mine = ++run;
		loading = true;
		error = '';
		try {
			const result = await fn();
			if (mine === run) data = result;
		} catch (e) {
			if (mine === run) error = errorMessage(e);
		} finally {
			if (mine === run) loading = false;
		}
	}
	$effect(() => {
		load();
	});

	return {
		get data() { return data; },
		get loading() { return loading; },
		get error() { return error; },
		reload: load
	};
}
