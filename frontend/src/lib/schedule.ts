// Ders programı: kayıtları klasik haftalık tabloya (satır = ders saati, sütun = gün) dönüştürür.
import type { Rec } from '$lib/api/types';

export const LESSON_MINUTES = 40;
export const BREAK_MINUTES = 10;

export type Lesson = { key: string; entryId: string; day: number; start: number; end: number; courseId: string; classId: string };
export type Slot = { start: number; end: number };
export type Timetable = { days: number[]; slots: Slot[]; lessons: Lesson[]; at: (day: number, start: number) => Lesson[] };

const WEEK = [1, 2, 3, 4, 5, 6, 0]; // Pazartesi … Cumartesi, Pazar
const WORKDAYS = [1, 2, 3, 4, 5];

/** Bir program kaydı (örn. 09:00–10:30) ders saatlerine bölünür: 40 dk ders + 10 dk teneffüs. */
export function expandEntry(entry: Rec, lesson = LESSON_MINUTES, gap = BREAK_MINUTES): Lesson[] {
	const start = Number(entry.start_minute);
	const end = Number(entry.end_minute);
	const day = Number(entry.day_of_week);
	if (!Number.isFinite(start) || !Number.isFinite(end) || !Number.isFinite(day) || end <= start || day < 0 || day > 6) return [];
	const slot = lesson + gap;
	const count = Math.max(1, Math.floor((end - start + gap) / slot));
	const result: Lesson[] = [];
	for (let i = 0; i < count; i++) {
		const lessonStart = start + i * slot;
		result.push({
			key: `${entry.id}-${i}`,
			entryId: String(entry.id),
			day,
			start: lessonStart,
			end: Math.min(lessonStart + lesson, end),
			courseId: String(entry.course_id ?? ''),
			classId: String(entry.class_id ?? '')
		});
	}
	return result;
}

export function buildTimetable(rows: Rec[], lesson = LESSON_MINUTES, gap = BREAK_MINUTES): Timetable {
	const lessons = rows.flatMap((row) => expandEntry(row, lesson, gap));
	const used = new Set(lessons.map((l) => l.day));
	const days = lessons.length ? WEEK.filter((d) => WORKDAYS.includes(d) || used.has(d)) : [];

	const ends = new Map<number, number>();
	for (const l of lessons) ends.set(l.start, Math.max(ends.get(l.start) ?? 0, l.end));
	const slots = [...ends.entries()].sort((a, b) => a[0] - b[0]).map(([start, end]) => ({ start, end }));

	return { days, slots, lessons, at: (day, start) => lessons.filter((l) => l.day === day && l.start === start) };
}
