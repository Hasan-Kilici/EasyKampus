import { api } from '$lib/api';
import { ApiError, errorMessage, token } from '$lib/api/client';
import type { User } from '$lib/api/types';

class Session {
	user = $state<User | null>(null);
	role = $derived(this.user?.role);
	isAdmin = $derived(this.role === 'admin' || this.role === 'superadmin');
	isTeacher = $derived(this.role === 'teacher');
	isStudent = $derived(this.role === 'student');
	canManage = $derived(this.isAdmin || this.isTeacher);

	error = $state('');

	async load() {
		this.error = '';
		if (!token.get()) return;
		try {
			this.user = await api.users.me();
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				token.clear(); // yalnızca geçersiz oturumda sil
				this.user = null;
			} else {
				// 429 / 500 / ağ hatası: oturumu silme, hata göster ve tekrar denemeye izin ver
				this.error = errorMessage(e);
			}
		}
	}

	async logout() {
		try {
			await api.auth.logout();
		} catch {
			/* token zaten geçersiz olabilir */
		}
		token.clear();
		this.user = null;
	}
}

export const session = new Session();
