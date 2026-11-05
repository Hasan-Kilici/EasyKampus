import type { Rec } from '$lib/api/types';

// day_of_week: Go time.Weekday varsayımı (0 = Pazar). Backend ile doğrula.
export const DAYS = ['Pazar', 'Pazartesi', 'Salı', 'Çarşamba', 'Perşembe', 'Cuma', 'Cumartesi'];
const pad = (n: number) => String(n).padStart(2, '0');
export const fmtTime = (m: number) => `${pad(Math.floor(m / 60))}:${pad(m % 60)}`;
export const toMinutes = (t: string) => { const [h, m] = t.split(':').map(Number); return h * 60 + m; };

export const STATUS: Record<string, string> = {
	excellent: 'Mükemmel', good: 'İyi', needs_attention: 'Dikkat', weak: 'Zayıf', unknown: 'Bilinmiyor',
	info: 'Bilgi', low: 'Düşük', medium: 'Orta', high: 'Yüksek',
	draft: 'Taslak', pending: 'Onay bekliyor', approved: 'Onaylı', rejected: 'Reddedildi',
	present: 'Var', absent: 'Yok', late: 'Geç', excused: 'Mazeretli',
	active: 'Aktif', withdrawn: 'Ayrıldı', completed: 'Tamamlandı', in_progress: 'Devam ediyor', abandoned: 'Yarım kaldı',
	exam: 'Sınav', assignment: 'Ödev', project: 'Proje', participation: 'Katılım',
	superadmin: 'Süper yönetici', admin: 'Yönetici', teacher: 'Öğretmen', student: 'Öğrenci',
	director: 'Müdür', registrar: 'Kayıt sorumlusu'
};

const LABELS: Record<string, string> = {
	name: 'Ad', score: 'Puan', status: 'Durum', attendance: 'Devam', activity: 'Aktivite', completion: 'Tamamlama',
	student_count: 'Öğrenci', class_count: 'Sınıf', improvement: 'Gelişim', average_student_score: 'Ort. puan',
	participation: 'Katılım', questions: 'Soru', correct: 'Doğru', type: 'Tür', term: 'Dönem', note: 'Not',
	date: 'Tarih', created_at: 'Oluşturulma', occurred_at: 'Zaman', duration_seconds: 'Süre (sn)', events: 'Olay',
	logins: 'Giriş', department_fit: 'Bölüm uyumu', relevance: 'Bölüm ilgisi', study_minutes: 'Çalışma (dk)', lesson_views: 'Ders görüntüleme', enrolled_at: 'Kayıt tarihi', completed_at: 'Tamamlanma'
};

export function fmt(key: string, v: unknown): string {
	if (v === null || v === undefined || v === '') return '—';
	if (typeof v === 'boolean') return v ? 'Evet' : 'Hayır';
	if (typeof v === 'number') return key.endsWith('_minute') ? fmtTime(v) : String(Math.round(v * 100) / 100);
	if (typeof v === 'string') {
		if (/^\d{4}-\d\d-\d\dT/.test(v)) return new Date(v).toLocaleString('tr-TR', { dateStyle: 'medium', timeStyle: 'short' });
		return STATUS[v] ?? v;
	}
	return JSON.stringify(v);
}

export const autoColumns = (rows: Rec[]) =>
	Object.keys(rows[0] ?? {})
		.filter((k) => k !== 'id' && !k.endsWith('_id') && typeof rows[0][k] !== 'object')
		.map((k) => ({ key: k, label: LABELS[k] ?? k }));

export const idMap = (rows: Rec[] = [], label: (r: Rec) => string = (r) => String(r.name)) =>
	new Map(rows.map((r) => [String(r.id), label(r)]));
export const personName = (r: Rec) => `${r.name} ${r.surname}`;
export const lookup = (m: Map<string, string>, id: unknown) => m.get(String(id)) ?? String(id ?? '—').slice(0, 8);
