// Package bootstrap connects the parts of the application: it builds the adapters and the use cases of
// each context, and the HTTP engine. The API command and the E2E tests both use it.
package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"

	adminapp "github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	adminhttp "github.com/rudradcruze/go-react-template/backend/internal/admin/http"
	admininfra "github.com/rudradcruze/go-react-template/backend/internal/admin/infra"
	examapp "github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	examhttp "github.com/rudradcruze/go-react-template/backend/internal/exam/http"
	examinfra "github.com/rudradcruze/go-react-template/backend/internal/exam/infra"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/config"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/server"
)

// NewAPI builds the HTTP handler of the API.
func NewAPI(cfg config.Config, log *slog.Logger, db *gorm.DB, cacheClient *redis.Client, _ *asynq.Client) http.Handler {
	admins := admininfra.NewAdminRepository(db)
	hasher := admininfra.NewBcryptHasher()
	sessions := admininfra.NewRedisSessionStore(cacheClient)
	failedLogins := admininfra.NewRedisFailedLogins(cacheClient)
	adminHandler := adminhttp.NewHandler(
		adminapp.NewLogin(admins, hasher, sessions, failedLogins),
		adminapp.NewLogout(sessions),
		adminapp.NewCurrentAdmin(sessions, admins),
		log,
	)

	questions := examinfra.NewQuestionRepository(db)
	exams := examinfra.NewExamRepository(db)
	attempts := examinfra.NewAttemptRepository(db)
	examHandler := examhttp.NewHandler(examhttp.UseCases{
		CreateQuestion:     examapp.NewCreateQuestion(questions, time.Now),
		GetQuestion:        examapp.NewGetQuestion(questions),
		ListQuestions:      examapp.NewListQuestions(questions),
		UpdateQuestion:     examapp.NewUpdateQuestion(questions),
		DeleteQuestion:     examapp.NewDeleteQuestion(questions),
		CreateExam:         examapp.NewCreateExam(exams, time.Now),
		GetExam:            examapp.NewGetExam(exams),
		ListExams:          examapp.NewListExams(exams),
		UpdateExam:         examapp.NewUpdateExam(exams, attempts),
		RegenerateExamLink: examapp.NewRegenerateExamLink(exams),
		DeleteExam:         examapp.NewDeleteExam(exams),
		GetExamLink:        examapp.NewGetExamLink(exams),
		StartAttempt:       examapp.NewStartAttempt(exams, questions, attempts, time.Now),
		GetAttempt:         examapp.NewGetAttempt(exams, questions, attempts, time.Now),
		SaveAnswer:         examapp.NewSaveAnswer(exams, questions, attempts, time.Now),
		SubmitAttempt:      examapp.NewSubmitAttempt(exams, questions, attempts, time.Now),
		ListExamAttempts:   examapp.NewListExamAttempts(exams, questions, attempts, time.Now),
	}, log)
	// The Admin routes of the exam context use the guard of the admin context. The exam context does not import it.
	examRoutes := func(api *gin.RouterGroup) { examHandler.Routes(api, adminHandler.RequireSession) }
	return server.New(cfg, log, adminHandler.Routes, examRoutes)
}

// NewSeedAdmin builds the use case that makes the first Admin. The migrate command runs it.
func NewSeedAdmin(db *gorm.DB) *adminapp.SeedAdmin {
	return adminapp.NewSeedAdmin(admininfra.NewAdminRepository(db), admininfra.NewBcryptHasher(), time.Now)
}

// NewWorkerMux builds the handlers of the worker: it routes each task type to its handler.
// The application has no background task now, so the mux has no handler.
func NewWorkerMux(_ *slog.Logger, _ *gorm.DB) *asynq.ServeMux {
	return asynq.NewServeMux()
}
