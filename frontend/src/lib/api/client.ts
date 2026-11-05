// Swagger: host localhost:3000, yollar "/api/..." ile başlıyor -> BASE = origin + "/api"
// Geliştirmede backend localhost:3000; canlıda (Vercel) aynı alan adı üzerinden "/api" kullanılır.
// VITE_API_URL tanımlıysa her ortamda o kazanır.
const ORIGIN: string = import.meta.env.VITE_API_URL ?? (import.meta.env.DEV ? 'http://localhost:3000' : '');
const BASE = `${ORIGIN}/api`;
const KEY = 'lms_token';

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string,
		message: string,
		public issues: { path: string; message: string }[] = []
	) {
		super(message);
	}
}

// "Beni hatırla" açıksa localStorage, değilse sessionStorage
export const token = {
	get: () => localStorage.getItem(KEY) ?? sessionStorage.getItem(KEY),
	set(value: string, remember: boolean) {
		this.clear();
		(remember ? localStorage : sessionStorage).setItem(KEY, value);
	},
	clear() {
		localStorage.removeItem(KEY);
		sessionStorage.removeItem(KEY);
	}
};

type Query = Record<string, string | number | undefined>;

// 401 gelince (ve token vardıysa) oturum bitmiş demektir: sadece bu durumda token silinir ve login'e gidilir
let onUnauthorized: (() => void) | null = null;
export const setUnauthorizedHandler = (fn: (() => void) | null) => { onUnauthorized = fn; };

// Aynı anda atılan birebir aynı GET istekleri tek istekte birleşir (arka arkaya istek patlamasını önler)
const inflight = new Map<string, Promise<unknown>>();

async function send<T>(
	method: 'GET' | 'POST' | 'PUT' | 'DELETE',
	path: string,
	o: { query?: Query; body?: unknown } = {}
): Promise<T> {
	const url = new URL(BASE + path, typeof location !== 'undefined' ? location.origin : 'http://localhost');
	for (const [k, v] of Object.entries(o.query ?? {}))
		if (v !== undefined && v !== '') url.searchParams.set(k, String(v));

	const t = token.get();
	const res = await fetch(url, {
		method,
		headers: {
			...(o.body !== undefined && { 'Content-Type': 'application/json' }),
			...(t && { Authorization: `Bearer ${t}` })
		},
		body: o.body !== undefined ? JSON.stringify(o.body) : undefined
	});

	if (res.status === 204) return undefined as T;
	const json = await res.json().catch(() => null);

	if (!res.ok) {
		if (res.status === 401 && t) { token.clear(); onUnauthorized?.(); }
		// Swagger: hata gövdesi { data, error: { code, message } }
		throw new ApiError(res.status, json?.error?.code ?? 'UNKNOWN', json?.error?.message ?? res.statusText, json?.error?.issues ?? []);
	}
	// Gerçek API her cevabı { data, error } zarfında döner (swagger zarfı göstermiyor)
	return (json && typeof json === 'object' && 'data' in json ? json.data : json) as T;
}

export function request<T = unknown>(
	method: 'GET' | 'POST' | 'PUT' | 'DELETE',
	path: string,
	o: { query?: Query; body?: unknown } = {}
): Promise<T> {
	if (method !== 'GET') return send<T>(method, path, o);
	const key = path + JSON.stringify(o.query ?? {});
	const hit = inflight.get(key);
	if (hit) return hit as Promise<T>;
	const p = send<T>(method, path, o).finally(() => inflight.delete(key));
	inflight.set(key, p);
	return p;
}

export function errorMessage(e: unknown): string {
	if (e instanceof ApiError) {
		if (e.status === 401) return 'Oturumun sona erdi. Tekrar giriş yap.';
		if (e.status === 403) return 'Bu işlem için yetkin yok.';
		if (e.status === 429) return 'Çok fazla istek gönderildi. Birkaç saniye bekleyip tekrar dene.';
		if (e.status === 503) return 'Analiz servisi şu an kapalı. Biraz sonra tekrar dene.';
		if (e.issues.length) return e.issues.map((i) => i.message).join(' ');
		return e.message;
	}
	if (e instanceof Error && !(e instanceof TypeError)) return e.message;
	return 'Sunucuya ulaşılamadı. Bağlantını kontrol et.';
}

export async function download(path: string, filename: string) {
	const t = token.get();
	const res = await fetch(`${BASE}${path}`, { headers: t ? { Authorization: `Bearer ${t}` } : {} });
	if (!res.ok) throw new ApiError(res.status, 'DOWNLOAD', res.statusText);
	const url = URL.createObjectURL(await res.blob());
	Object.assign(document.createElement('a'), { href: url, download: filename }).click();
	URL.revokeObjectURL(url);
}

// Presigned R2 yüklemesi: Authorization başlığı gönderilmez, istek imzaya dahil değildir.
export function uploadToUrl(
	url: string,
	file: Blob,
	contentType: string,
	onProgress?: (percent: number) => void
): Promise<void> {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('PUT', url);
		xhr.setRequestHeader('Content-Type', contentType);
		xhr.upload.onprogress = (event) => {
			if (event.lengthComputable && onProgress) {
				onProgress(Math.round((event.loaded / event.total) * 100));
			}
		};
		xhr.onload = () => {
			if (xhr.status >= 200 && xhr.status < 300) resolve();
			else reject(new ApiError(xhr.status, 'UPLOAD', xhr.statusText || 'Yükleme başarısız'));
		};
		xhr.onerror = () => reject(new ApiError(0, 'UPLOAD', 'Yükleme sırasında bağlantı hatası'));
		xhr.send(file);
	});
}
