import { mount, flushSync } from 'svelte';
import { test, expect, vi, beforeEach } from 'vitest';
import CrudPage from '$lib/components/CrudPage.svelte';
import { request, token, ApiError, setUnauthorizedHandler } from '$lib/api/client';
import { api } from '$lib/api';
import { session } from '$lib/session.svelte';

(HTMLDialogElement.prototype as any).showModal = function () { this.open = true; };
(HTMLDialogElement.prototype as any).close = function () { this.open = false; };
const wait = (ms = 120) => new Promise((r) => setTimeout(r, ms));
const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

beforeEach(() => { localStorage.clear(); sessionStorage.clear(); document.body.innerHTML = ''; setUnauthorizedHandler(null); vi.restoreAllMocks(); });

test('ekle akışı: create 1 kez, liste toplam 2 kez', async () => {
	const load = vi.fn(async () => [{ id: '1', name: 'a' }]);
	const create = vi.fn(async () => ({}));
	mount(CrudPage, { target: document.body, props: { title: 'T', columns: [{ key: 'name', label: 'Ad' }], fields: [{ name: 'm', label: 'M' }], load, create } });
	flushSync(); await wait();
	(document.querySelector('button.bg-\\[\\#facc15\\]') as HTMLButtonElement).click();
	flushSync(); await wait();
	const input = document.querySelector('dialog input') as HTMLInputElement;
	input.value = 'x'; input.dispatchEvent(new Event('input', { bubbles: true }));
	document.querySelector('dialog form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
	flushSync(); await wait(200);
	console.log('RESULT create:', create.mock.calls.length, 'load:', load.mock.calls.length, 'gövde:', JSON.stringify(create.mock.calls[0]));
	expect(create).toHaveBeenCalledTimes(1);
	expect(load).toHaveBeenCalledTimes(2);
});

test('zarf açılır, null liste [] olur', async () => {
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(200, { data: null, error: null }));
	expect(await api.courses.list()).toEqual([]);
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(200, { data: { token: 't', user: { id: '1' } }, error: null }));
	expect((await api.auth.login({ email: 'a', password: 'b' })).token).toBe('t');
});

test('429: oturum SİLİNMEZ, yönlendirme olmaz', async () => {
	token.set('abc', true);
	const handler = vi.fn(); setUnauthorizedHandler(handler);
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(429, { data: null, error: { code: 'RATE', message: 'rate limit' } }));
	await expect(request('GET', '/x')).rejects.toBeInstanceOf(ApiError);
	console.log('RESULT 429 sonrası token:', token.get(), 'handler:', handler.mock.calls.length);
	expect(token.get()).toBe('abc'); expect(handler).not.toHaveBeenCalled();
});

test('session.load 500/429 olursa token korunur', async () => {
	token.set('abc', false);
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(500, { data: null, error: { code: 'X', message: 'boom' } }));
	await session.load();
	console.log('RESULT session 500 sonrası token:', token.get(), 'error:', session.error);
	expect(token.get()).toBe('abc'); expect(session.error).toBe('boom'); expect(session.user).toBeNull();
});

test('401: token silinir ve login handler çağrılır', async () => {
	token.set('abc', true);
	const handler = vi.fn(); setUnauthorizedHandler(handler);
	vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(401, { data: null, error: { code: 'UNAUTH', message: 'no' } }));
	await expect(request('GET', '/x')).rejects.toBeInstanceOf(ApiError);
	expect(token.get()).toBeNull(); expect(handler).toHaveBeenCalledTimes(1);
});

test('aynı anda 5 aynı GET -> tek fetch', async () => {
	const f = vi.spyOn(globalThis, 'fetch').mockImplementation(async () => { await wait(20); return json(200, { data: [], error: null }); });
	await Promise.all([1, 2, 3, 4, 5].map(() => api.courses.list()));
	console.log('RESULT 5 çağrı -> fetch:', f.mock.calls.length);
	expect(f).toHaveBeenCalledTimes(1);
});
