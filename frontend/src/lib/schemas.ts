import { z } from 'zod';

const email = z
	.string()
	.trim() // Postel yasası: baştaki/sondaki boşluklar sessizce temizlenir
	.min(1, 'E-posta gir.')
	.email('Geçerli bir e-posta gir. Örnek: ad@okul.edu.tr');

export const loginSchema = z.object({
	email,
	password: z.string().min(1, 'Şifreni gir.'),
	remember: z.boolean()
});

export const registerSchema = z.object({
	name: z.string().trim().min(2, 'Adını gir.'),
	surname: z.string().trim().min(2, 'Soyadını gir.'),
	age: z.coerce.number().int('Geçerli bir yaş gir.').min(1, 'Geçerli bir yaş gir.').max(120, 'Geçerli bir yaş gir.'),
	email,
	password: z.string().min(8, 'Şifre en az 8 karakter olmalı.'),
	terms: z.boolean().refine((v) => v, 'Devam etmek için kullanım koşullarını kabul et.')
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;

/** Zod hatalarını alan adı -> ilk hata mesajı haritasına çevirir */
export function fieldErrors(error: z.ZodError): Record<string, string> {
	const out: Record<string, string> = {};
	for (const issue of error.issues) {
		const key = String(issue.path[0]);
		if (!out[key]) out[key] = issue.message;
	}
	return out;
}
