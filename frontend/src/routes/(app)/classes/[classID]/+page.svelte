<script lang="ts">
	import { page } from '$app/state';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Tabs from '$lib/components/Tabs.svelte';
	import CrudPage from '$lib/components/CrudPage.svelte';
	import FormDialog from '$lib/components/FormDialog.svelte';
	import BulkEnrollPanel from '$lib/components/BulkEnrollPanel.svelte';
	import ClassSchedule from '$lib/components/ClassSchedule.svelte';
	import BulkAttendancePanel from '$lib/components/BulkAttendancePanel.svelte';
	import BulkGradesPanel from '$lib/components/BulkGradesPanel.svelte';
	import { resource } from '$lib/resource.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { DAYS, idMap, lookup, personName } from '$lib/format';
	import { classOptions, courseOptions, classCourseOptions, classStudentOptions, userOptions, ATTENDANCE_OPTS, GRADE_TYPE_OPTS, DAY_OPTS, TERM_FIELD, type Field } from '$lib/options';
	import { ghost, input } from '$lib/ui';
	import type { Rec } from '$lib/api/types';

	const cid = $derived(page.params.classID!);
	const klass = resource(() => api.classes.get(cid) as Promise<Rec>);
	const people = resource(() => (api.users.list() as unknown as Promise<Rec[]>).catch(() => [] as Rec[]));
	const courses = resource(() => api.courses.list().catch(() => [] as Rec[]));
	const who = $derived(idMap(people.data ?? [], personName));
	const what = $derived(idMap(courses.data ?? []));
	const m = $derived(session.canManage);

	let tab = $state('students');
	let refresh = $state(0);
	let tOpen = $state(false);
	let tRow = $state<Rec | null>(null);
	let nOpen = $state(false);
	let nRow = $state<Rec | null>(null);
	let atFrom = $state('');
	let atTo = $state('');
	let atStatus = $state('');

	const tabs = [
		{ id: 'students', label: 'Öğrenciler' }, { id: 'teachers', label: 'Öğretmenler' }, { id: 'courses', label: 'Dersler' },
		{ id: 'schedule', label: 'Program' }, { id: 'attendance', label: 'Yoklama' }, { id: 'grades', label: 'Notlar' }, { id: 'notes', label: 'Öğretmen notları' }
	];
	const person = (key: string, label: string) => ({ key, label, format: (r: Rec) => lookup(who, r[key]) });
	const course = (label = 'Ders') => ({ key: 'course_id', label, format: (r: Rec) => lookup(what, r.course_id) });

	const fStudent: Field = { name: 'student_id', label: 'Öğrenci', type: 'select', options: userOptions('student'), required: true };
	const fRosterStudent: Field = { ...fStudent, options: () => classStudentOptions(cid)() };
	const fTeacher: Field = { name: 'teacher_id', label: 'Öğretmen', type: 'select', options: userOptions('teacher'), required: true };
	const fCourse: Field = { name: 'course_id', label: 'Ders', type: 'select', options: courseOptions, required: true };
	const fClassCourse: Field = { ...fCourse, options: () => classCourseOptions(cid)() };
	const fTerm = TERM_FIELD;
	const fNote: Field = { name: 'note', label: 'Not' };
	// Öğretmen notu (öğrenci hakkında serbest metin): /classes/:id/notes
	const noteFields: Field[] = [
		fRosterStudent,
		fClassCourse,
		{ name: 'body', label: 'Not', type: 'textarea', required: true },
		{ name: 'visible_to_student', label: 'Öğrenci görebilsin', type: 'checkbox' }
	];
</script>

<PageHeader title={String(klass.data?.name ?? 'Sınıf')} subtitle="Sınıf yönetimi">
	{#snippet actions()}<a class={ghost} href="/quizzes?class={cid}">Sınavlar</a>{/snippet}
</PageHeader>
<Tabs items={tabs} bind:value={tab} />

{#key `${tab}-${cid}`}
	{#if tab === 'students'}
		{#if m}<BulkEnrollPanel classId={cid} onSaved={() => refresh++} />{/if}
		{#key refresh}
			<CrudPage compact title="Kayıtlı öğrenciler" createLabel="Öğrenci kaydet"
				columns={[person('student_id', 'Öğrenci'), { key: 'course_id', label: 'Dersler', format: (r: Rec) => (r.course_id ? lookup(what, r.course_id) : 'Sınıfın tüm dersleri') }, { key: 'term', label: 'Dönem' }, { key: 'status', label: 'Durum' }, { key: 'enrolled_at', label: 'Kayıt tarihi' }]}
				fields={[fStudent, fTerm, fNote]}
				load={() => api.class(cid).enrollments.list()}
				create={m ? (v) => api.class(cid).enrollments.enroll(v) : undefined}
				remove={m ? (r) => api.class(cid).enrollments.withdraw(String(r.student_id)) : undefined}>
				{#snippet extra(row)}
					<a class={ghost} href="/students/{row.student_id}">Analiz</a>
					{#if m}<button class={ghost} onclick={() => { nRow = row; nOpen = true; }}>Not ekle</button>{/if}
					{#if m}<button class={ghost} onclick={() => { tRow = row; tOpen = true; }}>Nakil</button>{/if}
				{/snippet}
			</CrudPage>
		{/key}
	{:else if tab === 'teachers'}
		<CrudPage compact title="Sınıf öğretmenleri" createLabel="Öğretmen ata"
			columns={[person('teacher_id', 'Öğretmen'), course(), { key: 'term', label: 'Dönem' }, { key: 'active', label: 'Aktif' }]}
			fields={[fTeacher, fClassCourse, fTerm]}
			load={() => api.class(cid).teachers.list()}
			create={session.isAdmin ? (v) => api.class(cid).teachers.assign(v) : undefined}
			remove={session.isAdmin ? (r) => api.class(cid).teachers.remove(String(r.teacher_id)) : undefined} />
	{:else if tab === 'courses'}
		<CrudPage compact title="Sınıf dersleri" createLabel="Ders ata"
			columns={[course(), { key: 'term', label: 'Dönem' }, { key: 'active', label: 'Aktif' }]}
			fields={[fCourse, fTerm]}
			load={() => api.class(cid).courses.list()}
			create={session.isAdmin ? (v) => api.class(cid).courses.assign(v) : undefined}
			remove={session.isAdmin ? (r) => api.class(cid).courses.remove(String(r.course_id)) : undefined} />
	{:else if tab === 'schedule'}
		<!-- Her sınıf (şube) kendi ders programını burada düzenler; tablo kayıtlar değişince yenilenir. -->
		{#key refresh}
			<div class="mb-6"><ClassSchedule classId={cid} courses={what} /></div>
		{/key}
		<CrudPage compact title="Program kayıtları" createLabel="Ders saati ekle"
			columns={[{ key: 'day_of_week', label: 'Gün', format: (r) => DAYS[Number(r.day_of_week)] ?? '—' }, { key: 'start_minute', label: 'Başlangıç' }, { key: 'end_minute', label: 'Bitiş' }, course(), person('teacher_id', 'Öğretmen'), { key: 'term', label: 'Dönem' }]}
			fields={[{ name: 'day_of_week', label: 'Gün', type: 'select', options: DAY_OPTS, numeric: true, required: true }, { name: 'start_minute', label: 'Başlangıç', type: 'time', required: true }, { name: 'end_minute', label: 'Bitiş', type: 'time', required: true }, fClassCourse, fTeacher, fTerm]}
			load={() => api.class(cid).schedules.list()}
			create={m ? async (v) => { const r = await api.class(cid).schedules.create(v); refresh++; return r; } : undefined}
			update={m ? async (r, v) => { const x = await api.schedules.update(String(r.id), v); refresh++; return x; } : undefined}
			remove={m ? async (r) => { const x = await api.schedules.remove(String(r.id)); refresh++; return x; } : undefined} />
	{:else if tab === 'attendance'}
		{#if m}<BulkAttendancePanel classId={cid} onSaved={() => refresh++} />{/if}
		{#key refresh}
			{@const aQuery = { from: atFrom || undefined, to: atTo || undefined, status: atStatus || undefined }}
			<CrudPage compact title="Yoklama" createLabel="Yoklama gir"
				columns={[{ key: 'date', label: 'Tarih' }, person('student_id', 'Öğrenci'), course(), { key: 'status', label: 'Durum' }, { key: 'note', label: 'Not' }]}
				fields={[fRosterStudent, fClassCourse, { name: 'date', label: 'Tarih', type: 'date', required: true }, { name: 'status', label: 'Durum', type: 'select', options: ATTENDANCE_OPTS, required: true }, fNote]}
				load={() => api.class(cid).attendance.list(aQuery)}
				create={m ? (v) => api.class(cid).attendance.record([{ ...v, class_id: cid }]) : undefined}
				update={m ? (r, v) => api.attendance.update(String(r.id), { ...v, class_id: cid }) : undefined}
				remove={m ? (r) => api.attendance.remove(String(r.id)) : undefined}
			>
				{#snippet headerExtra()}
					<input class="{input} !h-11 w-40" type="date" aria-label="Başlangıç tarihi" bind:value={atFrom} />
					<input class="{input} !h-11 w-40" type="date" aria-label="Bitiş tarihi" bind:value={atTo} />
					<select class="{input} !h-11 w-44" aria-label="Durum filtresi" bind:value={atStatus}>
						<option value="">Tüm durumlar</option>
						{#each ATTENDANCE_OPTS as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
					</select>
					<a class={ghost} href={`/attendance?from=${atFrom}&to=${atTo}&status=${atStatus}`} target="_blank" rel="noreferrer">Özet</a>
				{/snippet}
			</CrudPage>
		{/key}
	{:else if tab === 'notes'}
		{#key refresh}
			<CrudPage compact title="Öğretmen notları" createLabel="Not ekle"
				columns={[person('student_id', 'Öğrenci'), course(), { key: 'body', label: 'Not' }, { key: 'visible_to_student', label: 'Öğrenci görür' }, person('author_id', 'Yazan'), { key: 'created_at', label: 'Tarih' }]}
				fields={noteFields}
				load={() => api.class(cid).notes.list()}
				create={m ? (v) => api.class(cid).notes.create({ ...v, class_id: cid }) : undefined}
				update={m ? (r, v) => api.notes.update(String(r.id), { ...v, class_id: cid }) : undefined}
				remove={m ? (r) => api.notes.remove(String(r.id)) : undefined}>
				{#snippet extra(row)}
					<a class={ghost} href="/students/{row.student_id}">Analiz</a>
				{/snippet}
			</CrudPage>
		{/key}
	{:else}
		{#if m}<BulkGradesPanel classId={cid} onSaved={() => refresh++} />{/if}
		{#key refresh}
			<CrudPage compact title="Notlar" createLabel="Not gir"
				columns={[person('student_id', 'Öğrenci'), course(), { key: 'type', label: 'Tür' }, { key: 'score', label: 'Puan' }, { key: 'term', label: 'Dönem' }, { key: 'note', label: 'Not' }]}
				fields={[fRosterStudent, fClassCourse, { name: 'type', label: 'Tür', type: 'select', options: GRADE_TYPE_OPTS, required: true }, { name: 'score', label: 'Puan', type: 'number', required: true }, fTerm, fNote]}
				load={() => api.class(cid).grades.list()}
				create={m ? (v) => api.class(cid).grades.record({ ...v, class_id: cid }) : undefined}
				update={m ? (r, v) => api.grades.update(String(r.id), { score: v.score as number, note: v.note as string }) : undefined}
				remove={m ? (r) => api.grades.remove(String(r.id)) : undefined} />
		{/key}
	{/if}
{/key}

<FormDialog bind:open={nOpen} title="Öğrenciye not ekle" submitLabel="Notu kaydet"
	fields={noteFields}
	initial={{ student_id: nRow?.student_id, course_id: nRow?.course_id }}
	onsubmit={async (v) => { await api.class(cid).notes.create({ ...v, class_id: cid }); refresh++; }} />

<FormDialog bind:open={tOpen} title="Başka sınıfa nakil" submitLabel="Nakil et"
	fields={[{ name: 'new_class_id', label: 'Yeni sınıf', type: 'select', options: classOptions, required: true }, fTerm, fNote]}
	onsubmit={async (v) => { await api.class(cid).enrollments.transfer({ ...v, student_id: tRow!.student_id, course_id: tRow!.course_id }); refresh++; }} />
