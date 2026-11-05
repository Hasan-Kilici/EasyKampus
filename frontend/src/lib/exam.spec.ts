import { describe, expect, it } from 'vitest';
import { checkDraft, formatClock, fromDetail, fromLocalInput, moveItem, newDraft, newOption, remainingMs, toInput, toLocalInput, totalPoints } from './exam';
import type { ExamDetail } from '$lib/api/types';

const validDraft = () => {
	const draft = newDraft('course-1');
	draft.name = 'Vize';
	draft.questions[0].question = '2+2?';
	draft.questions[0].options = [newOption('3'), newOption('4', true)];
	return draft;
};

describe('checkDraft', () => {
	it('geçerli bir sınavda eksik yoktur', () => expect(checkDraft(validDraft())).toEqual([]));

	it('sunucudaki kurallarla aynı eksikleri bulur', () => {
		const cases: [string, (d: ReturnType<typeof validDraft>) => void][] = [
			['ad boş', (d) => (d.name = ' ')],
			['ders yok', (d) => (d.course_id = '')],
			['soru yok', (d) => (d.questions = [])],
			['soru metni boş', (d) => (d.questions[0].question = '')],
			['tek seçenek', (d) => (d.questions[0].options = [newOption('3', true)])],
			['doğru yok', (d) => (d.questions[0].options[1].answer = false)],
			['iki doğru', (d) => (d.questions[0].options[0].answer = true)],
			['boş seçenek', (d) => (d.questions[0].options[0].content = '')],
			['aynı seçenek', (d) => (d.questions[0].options[0].content = '4')],
			['puan çok yüksek', (d) => (d.questions[0].points = 101)],
			['hak 0', (d) => (d.max_attempts = 0)],
			['süre çok uzun', (d) => (d.duration_minutes = 601)],
			['kapanış açılıştan önce', (d) => { d.open_at = '2026-10-26T10:00'; d.close_at = '2026-10-26T09:00'; }]
		];
		for (const [name, mutate] of cases) {
			const draft = validDraft();
			mutate(draft);
			expect(checkDraft(draft).length, name).toBeGreaterThan(0);
		}
	});

	it('mesajlar Türkçedir ve soru numarasını içerir', () => {
		const draft = validDraft();
		draft.questions[0].options[1].answer = false;
		const [issue] = checkDraft(draft);
		expect(issue.path).toBe('questions[0].options');
		expect(issue.message).toContain('Soru 1');
	});
});

describe('toInput / fromDetail', () => {
	it('kırpar, boş konuyu atar ve zamanı ISO yapar', () => {
		const draft = validDraft();
		draft.name = '  Vize  ';
		draft.open_at = '2026-10-26T09:00';
		const body = toInput(draft);
		expect(body.name).toBe('Vize');
		expect(body.questions[0]).not.toHaveProperty('topic_id');
		expect(body.open_at).toBe(new Date('2026-10-26T09:00').toISOString());
		expect(body.close_at).toBeNull();
	});

	it('sunucu cevabından taslağı geri kurar', () => {
		const detail = {
			quiz: { course_id: 'c1', name: 'A', description: '', open_at: '0001-01-01T00:00:00Z', close_at: null, duration_minutes: 30, max_attempts: 2 },
			questions: [{ topic_id: null, question: 'Q', points: 2, options: [{ content: 'x', answer: true }, { content: 'y', answer: false }] }],
			issues: [], attempt_count: 0, questions_editable: true
		} as unknown as ExamDetail;
		const draft = fromDetail(detail);
		expect(draft.open_at).toBe('');
		expect(draft.questions[0].options[0].answer).toBe(true);
		expect(totalPoints(draft)).toBe(2);
		expect(checkDraft(draft)).toEqual([]);
	});
});

describe('yardımcılar', () => {
	it('moveItem sınırları aşmaz', () => {
		expect(moveItem([1, 2, 3], 0, 1)).toEqual([2, 1, 3]);
		expect(moveItem([1, 2, 3], 0, -1)).toEqual([1, 2, 3]);
		expect(moveItem([1, 2, 3], 2, 3)).toEqual([1, 2, 3]);
	});

	it('0 puanlı soru 1 sayılır', () => {
		const draft = validDraft();
		draft.questions[0].points = 0;
		expect(totalPoints(draft)).toBe(1);
	});

	it('yerel tarih dönüşümü gidiş-dönüş tutarlıdır', () => {
		const iso = fromLocalInput('2026-10-26T14:30')!;
		expect(toLocalInput(iso)).toBe('2026-10-26T14:30');
		expect(toLocalInput(null)).toBe('');
		expect(toLocalInput('0001-01-01T00:00:00Z')).toBe('');
		expect(fromLocalInput('')).toBeNull();
	});

	it('kalan süre sunucu saatine göre hesaplanır', () => {
		// İstemci saati sunucudan 5 dk geride; bitiş sunucu saatine göre 10 dk sonra.
		const serverNow = Date.parse('2026-10-26T09:00:00Z');
		const clientNow = serverNow - 5 * 60_000;
		const deadline = '2026-10-26T09:10:00Z';
		expect(remainingMs(deadline, new Date(serverNow).toISOString(), clientNow, clientNow)).toBe(10 * 60_000);
		expect(remainingMs(deadline, new Date(serverNow).toISOString(), clientNow, clientNow + 4 * 60_000)).toBe(6 * 60_000);
		expect(remainingMs(deadline, new Date(serverNow).toISOString(), clientNow, clientNow + 99 * 60_000)).toBe(0);
		expect(remainingMs(null, new Date(serverNow).toISOString(), clientNow, clientNow)).toBeNull();
	});

	it('süre biçimi', () => {
		expect(formatClock(0)).toBe('00:00');
		expect(formatClock(61_000)).toBe('01:01');
		expect(formatClock(3_661_000)).toBe('1:01:01');
		expect(formatClock(999)).toBe('00:01');
	});
});
