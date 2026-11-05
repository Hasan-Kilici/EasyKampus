import { describe, expect, it } from 'vitest';
import { canAccess } from './access';
import { nav, navFor } from './nav';

describe('canAccess', () => {
	it('öğrenci yönetim sayfalarına giremez (URL ile de)', () => {
		for (const path of ['/users', '/users/123', '/schools', '/campuses', '/classes', '/classes/abc', '/quizzes/new', '/quizzes/c1/q1/edit', '/students/xyz']) {
			expect(canAccess(path, 'student'), path).toBe(false);
		}
	});

	it('öğrenci kendi sayfalarına girebilir', () => {
		for (const path of ['/dashboard', '/courses', '/courses/abc', '/schedule', '/grades', '/attendance', '/notes', '/performance', '/quizzes', '/quizzes/c1/q1', '/attempts/a1']) {
			expect(canAccess(path, 'student'), path).toBe(true);
		}
	});

	it('öğretmen okul/kampüs/kullanıcı yönetimine giremez ama sınıf, sınav oluşturma ve öğrenci ayrıntısına girer', () => {
		for (const path of ['/users', '/schools', '/campuses']) expect(canAccess(path, 'teacher'), path).toBe(false);
		for (const path of ['/classes', '/classes/c1', '/quizzes/new', '/quizzes/c1/q1/edit', '/students/s1', '/notes', '/performance']) {
			expect(canAccess(path, 'teacher'), path).toBe(true);
		}
	});

	it('yönetici her yönetim sayfasına girer, öğretmen notları/performans sayfası yalnızca öğretmen ve öğrenciye aittir', () => {
		for (const path of ['/users', '/users/1', '/schools', '/campuses', '/classes', '/quizzes', '/quizzes/new', '/students/s1']) {
			expect(canAccess(path, 'admin'), path).toBe(true);
			expect(canAccess(path, 'superadmin'), path).toBe(true);
		}
		expect(canAccess('/notes', 'admin')).toBe(false);
		expect(canAccess('/performance', 'admin')).toBe(false);
	});

	it('oturum yoksa erişim yok; bilinmeyen yollar engellenmez; sondaki eğik çizgi önemsizdir', () => {
		expect(canAccess('/dashboard', undefined)).toBe(false);
		expect(canAccess('/olmayan-sayfa', 'student')).toBe(true);
		expect(canAccess('/users/', 'student')).toBe(false);
		expect(canAccess('/', 'student')).toBe(true);
	});

	it('/quizzes/new ve /edit ile benzer yollar sınav çözme sayfasıyla karışmaz', () => {
		expect(canAccess('/quizzes/c1/q1', 'student')).toBe(true);
		expect(canAccess('/quizzes/c1/q1/edit', 'student')).toBe(false);
		expect(canAccess('/quizzes/newer', 'student')).toBe(true);
	});
});

describe('navFor', () => {
	it('menüdeki her öğe, aynı rol için URL koruması tarafından da izinlidir', () => {
		for (const role of ['superadmin', 'admin', 'teacher', 'student'] as const) {
			for (const item of navFor(role)) expect(canAccess(item.href, role), `${role} ${item.href}`).toBe(true);
		}
	});

	it('öğrenci menüsünde yönetim öğeleri yoktur, Sınavlar vardır', () => {
		const labels = navFor('student').map((n) => n.label);
		expect(labels).toEqual(expect.arrayContaining(['Panel', 'Derslerim', 'Ders programı', 'Sınavlar', 'Notlar', 'Devamsızlık']));
		for (const forbidden of ['Kullanıcılar', 'Okullar', 'Kampüsler', 'Sınıflar']) expect(labels).not.toContain(forbidden);
	});

	it('yönetici menüsünde Sınavlar da bulunur (onay için)', () => {
		expect(navFor('admin').map((n) => n.label)).toContain('Sınavlar');
	});

	it('her menü öğesinin yolu bir erişim kuralıyla karşılanır (menüde olup korumasız sayfa kalmaz)', () => {
		for (const item of nav) {
			const allowed = (['superadmin', 'admin', 'teacher', 'student'] as const).filter((r) => canAccess(item.href, r));
			expect(allowed.length, item.href).toBeGreaterThan(0);
			expect(allowed.sort(), item.href).toEqual([...item.roles].sort());
		}
	});
});
