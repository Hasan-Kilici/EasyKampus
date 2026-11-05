// Sınav ekranları için saf yardımcılar (test edilebilir, bileşenden bağımsız).
import type { ExamDetail, ExamInput, ExamIssue } from '$lib/api/types';

export const MAX_OPTIONS = 10;
export const MIN_OPTIONS = 2;

export type DraftOption = { key: string; content: string; answer: boolean };
export type DraftQuestion = { key: string; topic_id: string; question: string; points: number; options: DraftOption[] };
export type ExamDraft = {
	course_id: string; name: string; description: string; open_at: string; close_at: string;
	duration_minutes: number; max_attempts: number; questions: DraftQuestion[];
};

let counter = 0;
const key = () => `k${++counter}`;

export const newOption = (content = '', answer = false): DraftOption => ({ key: key(), content, answer });
export const newQuestion = (): DraftQuestion => ({
	key: key(), topic_id: '', question: '', points: 1, options: [newOption(), newOption()]
});
export const newDraft = (courseId = ''): ExamDraft => ({
	course_id: courseId, name: '', description: '', open_at: '', close_at: '', duration_minutes: 40, max_attempts: 1, questions: [newQuestion()]
});

// <input type="datetime-local"> değeri ↔ ISO (UTC)
export function toLocalInput(iso?: string | null): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime()) || d.getUTCFullYear() < 1971) return '';
	const p = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
export const fromLocalInput = (value: string): string | null => {
	if (!value) return null;
	const d = new Date(value);
	return Number.isNaN(d.getTime()) ? null : d.toISOString();
};

export function toInput(draft: ExamDraft): ExamInput {
	return {
		course_id: draft.course_id,
		name: draft.name.trim(),
		description: draft.description.trim(),
		open_at: fromLocalInput(draft.open_at),
		close_at: fromLocalInput(draft.close_at),
		duration_minutes: Number(draft.duration_minutes) || 0,
		max_attempts: Number(draft.max_attempts) || 0,
		questions: draft.questions.map((q) => ({
			...(q.topic_id ? { topic_id: q.topic_id } : {}),
			question: q.question.trim(),
			points: Number(q.points) || 0,
			options: q.options.map((o) => ({ content: o.content.trim(), answer: o.answer }))
		}))
	};
}

export function fromDetail(detail: ExamDetail): ExamDraft {
	const quiz = detail.quiz;
	return {
		course_id: quiz.course_id,
		name: quiz.name,
		description: quiz.description ?? '',
		open_at: toLocalInput(quiz.open_at),
		close_at: toLocalInput(quiz.close_at),
		duration_minutes: quiz.duration_minutes,
		max_attempts: quiz.max_attempts,
		questions: detail.questions.map((q) => ({
			key: key(),
			topic_id: q.topic_id ? String(q.topic_id) : '',
			question: q.question,
			points: q.points,
			options: q.options.map((o) => ({ key: key(), content: o.content, answer: o.answer }))
		}))
	};
}

export const totalPoints = (draft: ExamDraft) => draft.questions.reduce((sum, q) => sum + (Number(q.points) === 0 ? 1 : Number(q.points) || 0), 0);

export function moveItem<T>(items: T[], from: number, to: number): T[] {
	if (to < 0 || to >= items.length || from === to) return items;
	const copy = [...items];
	const [item] = copy.splice(from, 1);
	copy.splice(to, 0, item);
	return copy;
}

// Sunucudaki exams.Check ile aynı kurallar (anlık geri bildirim için). Yetkili kaynak sunucudur.
export function checkDraft(draft: ExamDraft): ExamIssue[] {
	const issues: ExamIssue[] = [];
	const add = (path: string, message: string) => issues.push({ path, message });
	if (!draft.name.trim()) add('name', 'Sınav adı boş olamaz.');
	if (!draft.course_id) add('course_id', 'Ders seçin.');
	const duration = Number(draft.duration_minutes);
	if (!(duration >= 0 && duration <= 600)) add('duration_minutes', 'Süre 0–600 dakika arasında olmalı (0 = süresiz).');
	const attempts = Number(draft.max_attempts);
	if (!(attempts >= 1 && attempts <= 20)) add('max_attempts', 'Deneme hakkı 1–20 arasında olmalı.');
	if (draft.open_at && draft.close_at && new Date(draft.close_at) < new Date(draft.open_at)) {
		add('close_at', 'Kapanış zamanı açılış zamanından önce olamaz.');
	}
	if (draft.questions.length === 0) add('questions', 'En az bir soru ekleyin.');
	draft.questions.forEach((q, index) => {
		const n = index + 1;
		const base = `questions[${index}]`;
		const points = Number(q.points);
		if (!(points >= 0 && points <= 100)) add(`${base}.points`, `Soru ${n}: puan 0–100 arasında olmalı.`);
		if (!q.question.trim()) add(`${base}.question`, `Soru ${n}: soru metni boş.`);
		if (q.options.length < MIN_OPTIONS) add(`${base}.options`, `Soru ${n}: en az ${MIN_OPTIONS} seçenek gerekli.`);
		const seen = new Set<string>();
		let correct = 0;
		q.options.forEach((o, position) => {
			const content = o.content.trim();
			if (!content) return add(`${base}.options[${position}].content`, `Soru ${n}: ${position + 1}. seçenek boş.`);
			if (seen.has(content.toLowerCase())) add(`${base}.options[${position}].content`, `Soru ${n}: aynı seçenek birden fazla kez yazılmış.`);
			seen.add(content.toLowerCase());
			if (o.answer) correct++;
		});
		if (correct !== 1) add(`${base}.options`, `Soru ${n}: tam olarak bir doğru seçenek işaretleyin.`);
	});
	return issues;
}

// Sunucu saati ile istemci saati arasındaki farkı hesaba katarak kalan süre (ms).
export function remainingMs(deadlineIso: string | null, serverTimeIso: string, fetchedAtMs: number, nowMs: number): number | null {
	if (!deadlineIso) return null;
	const offset = new Date(serverTimeIso).getTime() - fetchedAtMs; // sunucu − istemci
	return Math.max(0, new Date(deadlineIso).getTime() - (nowMs + offset));
}

export function formatClock(ms: number): string {
	const total = Math.max(0, Math.ceil(ms / 1000));
	const h = Math.floor(total / 3600);
	const m = Math.floor((total % 3600) / 60);
	const s = total % 60;
	const p = (n: number) => String(n).padStart(2, '0');
	return h > 0 ? `${h}:${p(m)}:${p(s)}` : `${p(m)}:${p(s)}`;
}
