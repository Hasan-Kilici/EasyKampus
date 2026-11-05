import { api } from '$lib/api';
import type { Rec } from '$lib/api/types';
import { DAYS, STATUS } from '$lib/format';

export type Opt = { value: string; label: string };

export type FieldType =
    | 'text'
    | 'number'
    | 'email'
    | 'password'
    | 'select'
    | 'textarea'
    | 'date'
    | 'datetime'
    | 'time'
    | 'checkbox';

export type Field = {
    name: string;
    label: string;
    type?: FieldType;
    required?: boolean;
    options?: Opt[] | ((form?: Rec) => Opt[] | Promise<Opt[]>);
    optionsLoader?: (form: Rec) => Promise<Opt[]>;
    disabled?: boolean;
    dependsOn?: string;
    onChange?: (value: string, form: Rec) => void | Promise<void>;
    placeholder?: string;
    numeric?: boolean;
    default?: string;
    requiredOnCreate?: boolean;
    requiredOnUpdate?: boolean;
};

export function currentTerm(): string {
    const now = new Date();
    const month = now.getMonth();
    const start = month < 8 ? now.getFullYear() - 1 : now.getFullYear();
    const season = month >= 1 && month <= 7 ? 'Bahar' : 'Güz';
    return `${start}-${start + 1} ${season}`;
}

const toOpts = (p: Promise<Rec[]>, label: (r: Rec) => string): Promise<Opt[]> =>
	p.then((rows) => rows.map((r) => ({ value: String(r.id), label: label(r) })));

export const schoolOptions = () => toOpts(api.schools.list(), (r) => String(r.name));
export const campusOptions = (input?: string | Rec) => {
    const schoolId =
        typeof input === 'string' ? input : (input?.school_id as string | undefined);
    return toOpts(
        api.campuses.list(schoolId ? { school_id: schoolId } : undefined),
        (r) => String(r.name)
    );
};
export const classOptions = () => toOpts(api.classes.list(), (r) => String(r.name));
export const courseOptions = () => toOpts(api.courses.list(), (r) => `${r.code} · ${r.name}`);
export const classCourseOptions = (classId: string) => async (): Promise<Opt[]> => {
	const [offered, all] = await Promise.all([
		api.class(classId).courses.list(),
		api.courses.list()
	]);
	const labels = new Map(all.map((row) => [String(row.id), `${row.code} · ${row.name}`]));
	return offered.map((row) => ({
		value: String(row.course_id),
		label: labels.get(String(row.course_id)) ?? String(row.course_id)
	}));
};
export const classStudentOptions = (classId: string) => async (): Promise<Opt[]> => {
	const [roster, all] = await Promise.all([
		api.class(classId).enrollments.list(),
		(api.users.list() as unknown as Promise<Rec[]>).catch(() => [] as Rec[])
	]);
	const labels = new Map(all.map((row) => [String(row.id), `${row.name} ${row.surname}`]));
	const seen = new Set<string>();
	const options: Opt[] = [];
	for (const row of roster) {
		if (row.status !== 'active') continue;
		const value = String(row.student_id);
		if (seen.has(value)) continue;
		seen.add(value);
		options.push({ value, label: labels.get(value) ?? value });
	}
	return options;
};
export const userOptions = (role?: string) => () =>
	toOpts(
		(api.users.list() as unknown as Promise<Rec[]>).then((u) => (role ? u.filter((x) => x.role === role) : u)),
		(r) => `${r.name} ${r.surname}`
	);

const fromMap = (m: string[]) => m.map((value) => ({ value, label: STATUS[value] ?? value }));
export const ROLE_OPTS = fromMap(['student', 'teacher', 'admin', 'superadmin']);
export const STAFF_OPTS = fromMap(['director', 'registrar']);
export const GRADE_TYPE_OPTS = fromMap(['exam', 'assignment', 'project', 'participation']);
export const ATTENDANCE_OPTS = fromMap(['present', 'absent', 'late', 'excused']);
export const DAY_OPTS = DAYS.map((label, i) => ({ value: String(i), label }));
export const TERM_FIELD: Field = {
	name: 'term',
	label: 'Dönem',
	required: true,
	default: currentTerm(),
	placeholder: currentTerm()
};
