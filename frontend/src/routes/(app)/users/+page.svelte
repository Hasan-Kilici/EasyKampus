<script lang="ts">
    import { ArrowRight } from '@lucide/svelte';
    import CrudPage from '$lib/components/CrudPage.svelte';
    import { api } from '$lib/api';
    import { ROLE_OPTS, schoolOptions, campusOptions, type Field } from '$lib/options';
    import { ghost } from '$lib/ui';
    import type { Rec } from '$lib/api/types';

    const fields: Field[] = [
        { name: 'name', label: 'Ad', required: true },
        { name: 'surname', label: 'Soyad', required: true },
        { name: 'age', label: 'Yaş', type: 'number' },
        { name: 'email', label: 'E-posta', type: 'email', required: true },
        {
            name: 'password',
            label: 'Şifre (değiştirmek için doldur)',
            type: 'password',
            requiredOnCreate: true
        },
        {
            name: 'role',
            label: 'Rol',
            type: 'select',
            options: ROLE_OPTS,
            required: true
        },
        {
            name: 'school_id',
            label: 'Okul',
            type: 'select',
            options: () => schoolOptions()
        },
        {
            name: 'campus_id',
            label: 'Kampüs',
            type: 'select',
            dependsOn: 'school_id',
            options: (values) => campusOptions(String(values?.school_id ?? ''))
        }
    ];
</script>

<CrudPage
    title="Kullanıcılar"
    subtitle="Öğrenci, öğretmen ve yönetici hesapları."
    columns={[
        { key: 'name', label: 'Ad' },
        { key: 'surname', label: 'Soyad' },
        { key: 'role', label: 'Rol' },
        { key: 'age', label: 'Yaş' },
        { key: 'created_at', label: 'Kayıt' }
    ]}
    {fields}
    load={() => api.users.list() as unknown as Promise<Rec[]>}
    createLabel="Yeni kullanıcı"
    create={(v) => api.users.create(v)}
    update={(r, v) => api.users.update(String(r.id), v)}
    remove={(r) => api.users.remove(String(r.id))}
>
    {#snippet extra(row)}
        <a class={ghost} href="/users/{row.id}">
            Detay
            <ArrowRight class="size-4" />
        </a>
    {/snippet}
</CrudPage>