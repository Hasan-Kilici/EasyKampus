// Ortak sınıflar: app.css'e bağımlı değil, Tailwind doğrudan bu dosyayı tarar.
// Klasik, sakin kurumsal görünüm: 1px kenarlık, hafif gölge, 8px yarıçap, mavi marka rengi.
export const label = 'mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300';

export const input =
	'h-10 w-full rounded-lg border border-zinc-300 bg-white px-3 text-sm text-zinc-900 outline-none transition placeholder:text-zinc-400 focus:border-brand-600 focus:ring-4 focus:ring-brand-600/20 disabled:cursor-not-allowed disabled:bg-zinc-100 disabled:opacity-70 data-[invalid]:border-red-600 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-50 dark:placeholder:text-zinc-500 dark:focus:border-brand-400 dark:disabled:bg-zinc-800';

// Çok satırlı alan: `input` sabit yükseklikli olduğundan ayrı bir belirteç.
export const textarea =
	'min-h-20 w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 outline-none transition placeholder:text-zinc-400 focus:border-brand-600 focus:ring-4 focus:ring-brand-600/20 disabled:cursor-not-allowed disabled:bg-zinc-100 disabled:opacity-70 data-[invalid]:border-red-600 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-50 dark:placeholder:text-zinc-500 dark:focus:border-brand-400 dark:disabled:bg-zinc-800';

export const error = 'mt-1.5 text-sm font-medium text-red-600 dark:text-red-400';

export const button =
	'inline-flex h-10 w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 text-sm font-semibold text-white shadow-sm outline-none transition hover:bg-brand-700 focus-visible:ring-4 focus-visible:ring-brand-600/30 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-60';

export const ghost =
	'inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-zinc-300 bg-white px-3.5 text-sm font-medium text-zinc-800 shadow-sm outline-none transition hover:bg-zinc-50 focus-visible:ring-4 focus-visible:ring-brand-600/30 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:hover:bg-zinc-800';
export const solid =
	'inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-brand-600 px-3.5 text-sm font-semibold text-white shadow-sm outline-none transition hover:bg-brand-700 focus-visible:ring-4 focus-visible:ring-brand-600/30 disabled:cursor-not-allowed disabled:opacity-60';
export const card =
	'rounded-lg border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-zinc-900';
// Dikkat çekilecek bilgi/uyarı kutusu (renk çakışması olmaması için ayrı belirteç).
export const notice =
	'rounded-lg border border-amber-300 bg-amber-50 p-5 text-zinc-900 dark:border-amber-800 dark:bg-amber-950/30 dark:text-zinc-50';
