import { describe, expect, it } from 'vitest';
import { buildTimetable, expandEntry } from './schedule';

const entry = (day: number, start: number, end: number, extra: Record<string, unknown> = {}) => ({
	id: `e${day}-${start}`, day_of_week: day, start_minute: start, end_minute: end, course_id: 'c1', class_id: 'k1', ...extra
});
const h = (hour: number, minute = 0) => hour * 60 + minute;

describe('expandEntry', () => {
	it('bloğu 40 dk ders + 10 dk teneffüse böler', () => {
		expect(expandEntry(entry(1, h(9), h(9, 40))).map((l) => [l.start, l.end])).toEqual([[h(9), h(9, 40)]]);
		expect(expandEntry(entry(1, h(9), h(9, 50))).length).toBe(1);
		expect(expandEntry(entry(1, h(9), h(10, 30))).map((l) => [l.start, l.end])).toEqual([[h(9), h(9, 40)], [h(9, 50), h(10, 30)]]);
		expect(expandEntry(entry(1, h(9), h(11, 30))).length).toBe(3);
	});

	it('kısa bloklar en az bir ders üretir ve bitişi aşmaz', () => {
		const [lesson] = expandEntry(entry(2, h(9), h(9, 30)));
		expect(lesson.end).toBe(h(9, 30));
	});

	it('geçersiz kayıtlar atlanır', () => {
		expect(expandEntry(entry(1, h(10), h(9)))).toEqual([]);
		expect(expandEntry(entry(9, h(9), h(10)))).toEqual([]);
		expect(expandEntry({ id: 'x', day_of_week: 'a', start_minute: 1, end_minute: 2 })).toEqual([]);
	});
});

describe('buildTimetable', () => {
	it('boş programda gün ve satır yoktur', () => {
		const t = buildTimetable([]);
		expect(t.days).toEqual([]);
		expect(t.slots).toEqual([]);
	});

	it('hafta içi günler her zaman, hafta sonu yalnızca dersi varsa görünür', () => {
		expect(buildTimetable([entry(1, h(9), h(9, 40))]).days).toEqual([1, 2, 3, 4, 5]);
		expect(buildTimetable([entry(6, h(9), h(9, 40))]).days).toEqual([1, 2, 3, 4, 5, 6]);
		expect(buildTimetable([entry(0, h(9), h(9, 40))]).days).toEqual([1, 2, 3, 4, 5, 0]);
	});

	it('satırlar başlangıç saatine göre sıralı ve tekrarsızdır', () => {
		const t = buildTimetable([entry(1, h(10), h(10, 40)), entry(2, h(9), h(9, 40)), entry(3, h(10), h(10, 40))]);
		expect(t.slots.map((s) => s.start)).toEqual([h(9), h(10)]);
	});

	it('aynı gün/saatteki farklı sınıflar aynı hücrede görünür', () => {
		const t = buildTimetable([entry(1, h(9), h(9, 40), { class_id: 'a', id: '1' }), entry(1, h(9), h(9, 40), { class_id: 'b', id: '2' })]);
		expect(t.at(1, h(9)).map((l) => l.classId).sort()).toEqual(['a', 'b']);
		expect(t.at(2, h(9))).toEqual([]);
	});

	it('çok saatlik blok ardışık satırlara yayılır', () => {
		const t = buildTimetable([entry(3, h(9), h(11, 30))]);
		expect(t.slots.map((s) => s.start)).toEqual([h(9), h(9, 50), h(10, 40)]);
		expect(t.at(3, h(9, 50))).toHaveLength(1);
	});
});
