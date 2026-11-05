import {
	LayoutDashboard, BookOpen, CalendarDays, FileQuestion, Award, ClipboardCheck,
	NotebookPen, BarChart3, Users, School, Building2, GraduationCap
} from '@lucide/svelte';
import type { Component } from 'svelte';
import type { Role } from '$lib/api/types';
import { ALL, STAFF, canAccess } from '$lib/access';

export interface NavItem { label: string; href: string; icon: Component<any>; roles: Role[]; hint: string }

const all = ALL;
const staff = STAFF;

export const nav: NavItem[] = [
	{ label: 'Panel', href: '/dashboard', icon: LayoutDashboard, roles: all, hint: 'Genel bakış' },
	{ label: 'Derslerim', href: '/courses', icon: BookOpen, roles: all, hint: 'Ders içerikleri ve videolar' },
	{ label: 'Ders programı', href: '/schedule', icon: CalendarDays, roles: all, hint: 'Haftalık ders tablosu' },
	{ label: 'Sınavlar', href: '/quizzes', icon: FileQuestion, roles: all, hint: 'Sınav çöz, oluştur ve onayla' },
	{ label: 'Notlar', href: '/grades', icon: Award, roles: all, hint: 'Sınav ve ödev notları' },
	{ label: 'Devamsızlık', href: '/attendance', icon: ClipboardCheck, roles: all, hint: 'Yoklama kayıtları' },
	{ label: 'Öğretmen notları', href: '/notes', icon: NotebookPen, roles: ['teacher', 'student'], hint: 'Öğretmen geri bildirimleri' },
	{ label: 'Performans', href: '/performance', icon: BarChart3, roles: ['teacher', 'student'], hint: 'Analiz ve öneriler' },
	{ label: 'Sınıflar', href: '/classes', icon: GraduationCap, roles: ['teacher', ...staff], hint: 'Sınıf ve şube yönetimi' },
	{ label: 'Okullar', href: '/schools', icon: School, roles: staff, hint: 'Okul tanımları' },
	{ label: 'Kampüsler', href: '/campuses', icon: Building2, roles: staff, hint: 'Kampüs tanımları' },
	{ label: 'Kullanıcılar', href: '/users', icon: Users, roles: staff, hint: 'Hesap yönetimi' }
];

/** Rolün menüde görebileceği öğeler (menü ile URL koruması aynı kuraldan beslenir). */
export const navFor = (role: Role | undefined): NavItem[] =>
	nav.filter((n) => role !== undefined && n.roles.includes(role) && canAccess(n.href, role));
