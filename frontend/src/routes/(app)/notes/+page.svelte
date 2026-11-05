<script lang="ts">
	import CrudPage from '$lib/components/CrudPage.svelte';
	import { api } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { classOptions, courseOptions, userOptions, type Field } from '$lib/options';

	const m = $derived(session.canManage);
	const fields: Field[] = [
		{ name: 'class_id', label: 'Sınıf', type: 'select', options: classOptions, required: true },
		{ name: 'course_id', label: 'Ders', type: 'select', options: courseOptions, required: true },
		{ name: 'student_id', label: 'Öğrenci', type: 'select', options: userOptions('student'), required: true },
		{ name: 'body', label: 'Not', type: 'textarea', required: true },
		{ name: 'visible_to_student', label: 'Öğrenci görebilsin', type: 'checkbox' }
	];
</script>

<CrudPage
	title="Notlarım"
	subtitle={session.isStudent ? 'Öğretmenlerinin seninle paylaştığı notlar.' : 'Öğrenciler hakkında tuttuğun notlar.'}
	columns={[{ key: 'body', label: 'Not' }, { key: 'visible_to_student', label: 'Öğrenci görür' }, { key: 'created_at', label: 'Tarih' }]}
	{fields}
	load={() => api.notes.list()}
	createLabel="Yeni not"
	create={m ? (v) => api.class(String(v.class_id)).notes.create(v) : undefined}
	update={m ? (r, v) => api.notes.update(String(r.id), v) : undefined}
	remove={m ? (r) => api.notes.remove(String(r.id)) : undefined}
/>
