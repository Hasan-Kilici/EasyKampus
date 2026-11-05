import { mount, flushSync, tick } from 'svelte';
import { test, expect, vi } from 'vitest';
import FormDialog from '$lib/components/FormDialog.svelte';
import CrudPage from '$lib/components/CrudPage.svelte';

(HTMLDialogElement.prototype as any).showModal = function () { this.open = true; };
(HTMLDialogElement.prototype as any).close = function () { this.open = false; };
const errs: string[] = [];
console.error = (...a: any[]) => { errs.push(String(a[0]?.stack ?? a[0]).split('\n').slice(0, 3).join(' | ')); };
const wait = (ms = 150) => new Promise((r) => setTimeout(r, ms));

test('FormDialog: seçenekler bir kez yüklenmeli', async () => {
	const spy = vi.fn(async () => [{ value: '1', label: 'a' }]);
	mount(FormDialog, { target: document.body, props: { open: true, title: 't', fields: [{ name: 'x', label: 'X', type: 'select', options: spy }], onsubmit: async () => {} } });
	flushSync(); await wait();
	console.log('RESULT FormDialog options çağrısı:', spy.mock.calls.length, '| hatalar:', errs.length, errs[0]);
	expect(spy.mock.calls.length).toBe(1);
});

test('CrudPage: liste ve seçenek istekleri sayısı', async () => {
	document.body.innerHTML = '';
	const load = vi.fn(async () => [{ id: '1', name: 'a' }]);
	const opts = vi.fn(async () => [{ value: '1', label: 'a' }]);
	const create = vi.fn(async () => ({}));
	mount(CrudPage, { target: document.body, props: { title: 'T', columns: [{ key: 'name', label: 'Ad' }], fields: [{ name: 'n', label: 'N', type: 'select', options: opts }, { name: 'm', label: 'M' }], load, create } });
	flushSync(); await wait();
	console.log('RESULT açılışta load:', load.mock.calls.length);
	(document.querySelector('button.bg-\\[\\#facc15\\]') as HTMLButtonElement)?.click();
	flushSync(); await wait();
	console.log('RESULT dialog açılınca opts:', opts.mock.calls.length, 'load:', load.mock.calls.length, '| hatalar:', errs.length, errs[0]);
	expect(opts.mock.calls.length).toBe(1);
});
