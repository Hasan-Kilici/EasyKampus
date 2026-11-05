// Sayfa erişimi için tek kaynak: hem yan menü hem URL koruması buradan beslenir.
// Asıl yetkilendirme sunucudadır (IDOR korumaları); bu yalnızca arayüzü rolle tutarlı kılar
// ve yetkisiz sayfaların boş/hatalı açılmasını engeller.
import type { Role } from '$lib/api/types';

export const ALL: Role[] = ['superadmin', 'admin', 'teacher', 'student'];
export const STAFF: Role[] = ['superadmin', 'admin'];
export const MANAGERS: Role[] = ['superadmin', 'admin', 'teacher'];

type Rule = { pattern: RegExp; roles: Role[] };

// İlk eşleşen kural kazanır: özel yollar genel olanlardan önce gelir.
const RULES: Rule[] = [
	{ pattern: /^\/quizzes\/new$/, roles: MANAGERS },
	{ pattern: /^\/quizzes\/[^/]+\/[^/]+\/edit$/, roles: MANAGERS },
	{ pattern: /^\/quizzes(\/|$)/, roles: ALL },
	{ pattern: /^\/attempts(\/|$)/, roles: ALL },
	{ pattern: /^\/dashboard$/, roles: ALL },
	{ pattern: /^\/courses(\/|$)/, roles: ALL },
	{ pattern: /^\/schedule$/, roles: ALL },
	{ pattern: /^\/grades$/, roles: ALL },
	{ pattern: /^\/attendance$/, roles: ALL },
	{ pattern: /^\/notes$/, roles: ['teacher', 'student'] },
	{ pattern: /^\/performance$/, roles: ['teacher', 'student'] },
	{ pattern: /^\/classes(\/|$)/, roles: ['teacher', ...STAFF] },
	// Öğrenci ayrıntısı (performans, notlar, devamsızlık): sunucu yalnızca kendi sınıfındaki öğrenciye izin verir.
	{ pattern: /^\/students\/[^/]+$/, roles: MANAGERS },
	{ pattern: /^\/(schools|campuses)(\/|$)/, roles: STAFF },
	{ pattern: /^\/users(\/|$)/, roles: STAFF }
];

/** Rol bu yola girebilir mi? Bilinmeyen yollar (404) engellenmez. */
export function canAccess(pathname: string, role: Role | undefined): boolean {
	if (!role) return false;
	const path = pathname.length > 1 ? pathname.replace(/\/+$/, '') : pathname;
	const rule = RULES.find((r) => r.pattern.test(path));
	return rule ? rule.roles.includes(role) : true;
}

export const HOME = '/dashboard';
