import { request as r, download } from './client';
import type * as T from './types';

type Q = Record<string, string | number | undefined>;
type Body = T.Rec; // formlar dinamik üretildiği için gövdeler gevşek; yanıtlar tam tipli

const crud = <Item>(base: string) => ({
	list: (query?: Q) => rl<Item>(base, { query }),
	get: (id: string) => r<Item>('GET', `${base}/${id}`),
	create: (body: Body) => r<Item>('POST', base, { body }),
	update: (id: string, body: Body) => r<Item>('PUT', `${base}/${id}`, { body }),
	remove: (id: string) => r<void>('DELETE', `${base}/${id}`)
});

// Boş listelerde backend data: null dönebilir -> her zaman dizi ver
const rl = <I>(path: string, o?: { query?: Q }) => r<I[] | null>('GET', path, o).then((x) => x ?? []);

const review = (path: string) => (body: T.ReviewInput) => r<unknown>('POST', `${path}/review`, { body });

export const api = {
	auth: {
		login: (body: T.LoginInput) => r<T.LoginResponse>('POST', '/auth/login', { body }),
		register: (body: T.RegisterInput) => r<T.Rec>('POST', '/auth/register', { body }),
		logout: () => r<void>('POST', '/auth/logout')
	},
	users: { ...crud<T.User>('/users'), me: () => r<T.User>('GET', '/users/me') },
	schools: {
		...crud<T.School>('/schools'),
		start: (body: T.StartSchoolInput) => r<T.StartSchoolResult>('POST', '/schools/start', { body }),
		staff: (id: string) => ({
			list: () => rl<T.SchoolStaff>(`/schools/${id}/staff`),
			add: (body: T.StaffInput) => r<T.Rec>('POST', `/schools/${id}/staff`, { body }),
			remove: (userId: string) => r<void>('DELETE', `/schools/${id}/staff/${userId}`)
		})
	},
	campuses: crud<T.Campus>('/campuses'),
	classes: crud<T.Class>('/classes'),
	courses: crud<T.Course>('/courses'),
	schedules: crud<T.ScheduleEntry>('/schedules'),
	notes: crud<T.Note>('/notes'),
	grades: {
		list: (query?: Q) => rl<T.Grade>('/grades', { query }),
		update: (id: string, body: T.GradeUpdate) => r<T.Grade>('PUT', `/grades/${id}`, { body }),
		remove: (id: string) => r<void>('DELETE', `/grades/${id}`),
		audit: (id: string) => rl<T.GradeAudit>(`/grades/${id}/audit`),
		export: () => download('/grades/export', 'notlar.csv')
		// TODO: import (text/csv) yükleme akışı
	},
	attendance: {
		list: (query?: Q) => rl<T.Attendance>('/attendance', { query }),
		summary: (query?: Q) => r<T.AttendancePeriodSummary>('GET', '/attendance/summary', { query }),
		update: (id: string, body: Body) => r<T.Attendance>('PUT', `/attendance/${id}`, { body }),
		remove: (id: string) => r<void>('DELETE', `/attendance/${id}`)
	},
	attempts: {
		list: (query?: Q) => rl<T.QuizAttempt>('/attempts', { query }),
		// swagger cevabı QuizScope gösteriyor ama gerçekte QuizAttempt dönmesi bekleniyor -> ikisini de karşılar
		get: (id: string) => r<Partial<T.QuizAttempt> & Partial<T.QuizScope> & T.Rec>('GET', `/attempts/${id}`),
		start: (quizId: string) => r<T.QuizAttempt>('POST', `/attempts/quizzes/${quizId}/start`),
		answer: (id: string, body: T.AnswerInput) => r<T.AttemptAnswer>('PUT', `/attempts/${id}/answers`, { body }),
		submit: (id: string) => r<T.QuizAttempt>('POST', `/attempts/${id}/submit`),
		// Sınav ekranı (cevap anahtarsız) ve sonuç
		exam: (id: string) => r<T.TakeExam>('GET', `/attempts/${id}/exam`),
		result: (id: string) => r<T.ExamResult>('GET', `/attempts/${id}/result`)
	},
	students: {
		me: {
			enrollments: () => rl<T.Enrollment>('/students/me/enrollments'),
			lessons: (courseId?: string) => rl<T.LessonCompletion>('/students/me/lessons', { query: { course_id: courseId } }),
			activities: (type?: string) => rl<T.Activity>('/students/me/activities', { query: { type } }),
			dailyActivities: () => rl<T.DailyAggregate>('/students/me/activities/daily'),
			performance: () => r<T.StudentPerformance>('GET', '/students/me/performance') // 503: analiz servisi kapalı olabilir
		}
	},
	student: (id: string) => ({
		profile: () => r<T.Rec>('GET', `/students/${id}/profile`),
		performance: () => r<T.StudentPerformance>('GET', `/students/${id}/performance`),
		activities: () => rl<T.Activity>(`/students/${id}/activities`),
		daily: () => rl<T.DailyAggregate>(`/students/${id}/activities/daily`),
		attendance: () => rl<T.Attendance>(`/students/${id}/attendance`),
		enrollments: () => rl<T.Enrollment>(`/students/${id}/enrollments`),
		grades: () => rl<T.Grade>(`/students/${id}/grades`),
		lessons: () => rl<T.LessonCompletion>(`/students/${id}/lessons`),
		notes: () => rl<T.Note>(`/students/${id}/notes`)
	}),
	teachers: {
		mePerformance: () => r<T.TeacherPerformance>('GET', '/teachers/me/performance'),
		performance: (id: string) => r<T.TeacherPerformance>('GET', `/teachers/${id}/performance`)
	},
	// --- iç içe kaynaklar ---
	course: (courseId: string) => ({
		lessons: {
			...crud<T.Lesson>(`/courses/${courseId}/lessons`),
			view: (id: string) => r<T.Lesson>('POST', `/courses/${courseId}/lessons/${id}/view`),
			complete: (id: string) => r<T.LessonCompletion>('POST', `/courses/${courseId}/lessons/${id}/complete`),
			review: (id: string) => review(`/courses/${courseId}/lessons/${id}`)
		},
		// swagger topics listesini tanımsız object bırakmış; dizi dönmesi bekleniyor
		topics: crud<T.Topic>(`/courses/${courseId}/topics`),
		// EFOR / AKTS ve konu–bölüm ilgisi (TF-IDF)
		workload: (hoursPerEcts?: number) =>
			r<T.Workload>('GET', `/courses/${courseId}/workload`, { query: { hours_per_ects: hoursPerEcts } }),
		contentAnalysis: () => r<T.ContentAnalysis>('GET', `/courses/${courseId}/content-analysis`),
		videos: (lessonId: string) => ({
			list: (query?: Q) => rl<T.Video>(`/courses/${courseId}/lessons/${lessonId}/videos`, { query }),
			get: (id: string) => r<T.Video>('GET', `/courses/${courseId}/lessons/${lessonId}/videos/${id}`),
			create: (body: T.VideoInput) =>
				r<T.Video>('POST', `/courses/${courseId}/lessons/${lessonId}/videos`, { body }),
			update: (id: string, body: Partial<T.VideoInput>) =>
				r<T.Video>('PUT', `/courses/${courseId}/lessons/${lessonId}/videos/${id}`, { body }),
			remove: (id: string) => r<void>('DELETE', `/courses/${courseId}/lessons/${lessonId}/videos/${id}`),
			presign: (body: T.PresignInput) =>
				r<T.PresignResult>('POST', `/courses/${courseId}/lessons/${lessonId}/videos/presign`, { body }),
			play: (id: string) =>
				r<T.PlaybackResult>('GET', `/courses/${courseId}/lessons/${lessonId}/videos/${id}/play`),
			complete: (videoId: string) =>
				r<{ completed: boolean }>('POST', `/courses/${courseId}/lessons/${lessonId}/videos/completions`, {
					body: { video_id: videoId }
				}),
			completions: (query?: Q) =>
				rl<T.VideoCompletion>(`/courses/${courseId}/lessons/${lessonId}/videos/completions`, { query })
		})
	}),
	class: (classId: string) => ({
		enrollments: {
			list: () => rl<T.Enrollment>(`/classes/${classId}/enrollments`),
			enroll: (body: Body) => r<T.Enrollment>('POST', `/classes/${classId}/enrollments`, { body }),
			bulk: (body: T.EnrollInput[]) => r<T.Enrollment[]>('POST', `/classes/${classId}/enrollments/bulk`, { body }),
			withdraw: (studentId: string) => r<void>('DELETE', `/classes/${classId}/enrollments/${studentId}`),
			transfer: (body: Body) => r<T.Enrollment>('POST', `/classes/${classId}/transfers`, { body })
		},
		teachers: {
			list: () => rl<T.TeacherAssignment>(`/classes/${classId}/teachers`),
			assign: (body: Body) => r<T.TeacherAssignment>('POST', `/classes/${classId}/teachers`, { body }),
			remove: (teacherId: string) => r<void>('DELETE', `/classes/${classId}/teachers/${teacherId}`)
		},
		courses: {
			list: () => rl<T.ClassCourse>(`/classes/${classId}/courses`),
			assign: (body: Body) => r<T.ClassCourse>('POST', `/classes/${classId}/courses`, { body }),
			remove: (courseId: string) => r<void>('DELETE', `/classes/${classId}/courses/${courseId}`)
		},
		grades: {
			list: (query?: Q) => rl<T.Grade>(`/classes/${classId}/grades`, { query }),
			record: (body: Body) => r<T.Grade>('POST', `/classes/${classId}/grades`, { body }),
			bulk: (body: T.GradeInput[]) => r<T.Grade[]>('POST', `/classes/${classId}/grades/bulk`, { body })
		},
		attendance: {
			list: (query?: Q) => rl<T.Attendance>(`/classes/${classId}/attendance`, { query }),
			record: (body: Body[]) => r<T.Attendance[]>('POST', `/classes/${classId}/attendance`, { body })
		},
		notes: crud<T.Note>(`/classes/${classId}/notes`),
		schedules: crud<T.ScheduleEntry>(`/classes/${classId}/schedules`),
		attempts: { list: (query?: Q) => rl<T.QuizAttempt>(`/classes/${classId}/attempts`, { query }) },
		// Sınav iş akışı: sınav + sorular + seçenekler tek işlemde
		exams: { create: (body: T.ExamInput) => r<T.ExamDetail>('POST', `/classes/${classId}/exams`, { body }) },
		exam: (quizId: string) => ({
			get: () => r<T.ExamDetail>('GET', `/classes/${classId}/exams/${quizId}`),
			replace: (body: T.ExamInput) => r<T.ExamDetail>('PUT', `/classes/${classId}/exams/${quizId}`, { body }),
			review: (body: { action: T.ReviewAction; note?: string }) => r<T.ExamDetail>('POST', `/classes/${classId}/exams/${quizId}/review`, { body }),
			results: () => r<T.ExamSummary>('GET', `/classes/${classId}/exams/${quizId}/results`)
		}),
		quizzes: {
			...crud<T.Quiz>(`/classes/${classId}/quizzes`),
			review: (id: string) => review(`/classes/${classId}/quizzes/${id}`)
		},
		questions: (quizId: string) => ({
			...crud<T.Question>(`/classes/${classId}/quizzes/${quizId}/questions`),
			review: (id: string) => review(`/classes/${classId}/quizzes/${quizId}/questions/${id}`),
			options: (questionId: string) => crud<T.Option>(`/classes/${classId}/quizzes/${quizId}/questions/${questionId}/options`)
		})
	})
};
