// The URLs of the API that the tests answer. MSW matches the path against the page origin.
export const sessionUrl = "/api/v1/admin/session";
export const questionsUrl = "/api/v1/admin/questions";
export const questionUrl = "/api/v1/admin/questions/:id";
export const examsUrl = "/api/v1/admin/exams";
export const examUrl = "/api/v1/admin/exams/:id";
export const examLinkUrl = "/api/v1/admin/exams/:id/exam-link";
export const examAttemptsUrl = "/api/v1/admin/exams/:id/attempts";
export const examLinkInfoUrl = "/api/v1/exam-links/:token";
export const startAttemptUrl = "/api/v1/exam-links/:token/attempts";
export const attemptUrl = "/api/v1/attempts/:id";
export const answerUrl = "/api/v1/attempts/:id/answers/:questionId";
export const submissionUrl = "/api/v1/attempts/:id/submission";
