// Package examhttp is the HTTP adapter of the Exam context.
package examhttp

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/problem"
)

// UseCases are the use cases that the handler calls.
type UseCases struct {
	CreateQuestion     *app.CreateQuestion
	GetQuestion        *app.GetQuestion
	ListQuestions      *app.ListQuestions
	UpdateQuestion     *app.UpdateQuestion
	DeleteQuestion     *app.DeleteQuestion
	CreateExam         *app.CreateExam
	GetExam            *app.GetExam
	ListExams          *app.ListExams
	UpdateExam         *app.UpdateExam
	RegenerateExamLink *app.RegenerateExamLink
	DeleteExam         *app.DeleteExam
	GetExamLink        *app.GetExamLink
	StartAttempt       *app.StartAttempt
	GetAttempt         *app.GetAttempt
	SaveAnswer         *app.SaveAnswer
	SubmitAttempt      *app.SubmitAttempt
	ListExamAttempts   *app.ListExamAttempts
}

// Handler serves the routes of the Exam context.
type Handler struct {
	uc  UseCases
	log *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(uc UseCases, log *slog.Logger) *Handler {
	return &Handler{uc: uc, log: log}
}

// Routes adds the routes of the context to the group of the API. requireAdmin is the guard of the Admin routes.
func (h *Handler) Routes(api *gin.RouterGroup, requireAdmin gin.HandlerFunc) {
	questions := api.Group("/admin/questions", requireAdmin)
	questions.POST("", h.postQuestion)
	questions.GET("", h.getQuestions)
	questions.GET("/:id", h.getQuestionByID)
	questions.PUT("/:id", h.putQuestion)
	questions.DELETE("/:id", h.removeQuestion)

	exams := api.Group("/admin/exams", requireAdmin)
	exams.POST("", h.postExam)
	exams.GET("", h.getExams)
	exams.GET("/:id", h.getExamByID)
	exams.PUT("/:id", h.putExam)
	exams.DELETE("/:id", h.removeExam)
	exams.POST("/:id/exam-link", h.postExamLink)
	exams.GET("/:id/attempts", h.getExamAttempts)

	// The routes of a Candidate need no session. The Exam link and the Attempt ID are the secrets.
	api.GET("/exam-links/:token", h.getExamLink)
	api.POST("/exam-links/:token/attempts", h.postAttempt)
	api.GET("/attempts/:id", h.getAttemptByID)
	api.PUT("/attempts/:id/answers/:questionId", h.putAnswer)
	api.POST("/attempts/:id/submission", h.postSubmission)
}

// parseID reads an ID from the address. It writes the problem detail and returns false when the ID is not valid.
func parseID(c *gin.Context, what string) (uuid.UUID, bool) {
	raw := c.Param("id")
	// Accept only the canonical form with dashes. uuid.Parse also accepts other forms, for example with braces.
	id, err := uuid.Parse(raw)
	if err != nil || id.String() != strings.ToLower(raw) {
		problem.Write(c, http.StatusBadRequest, fmt.Sprintf(
			"The ID is not valid. Use the ID of %s, in the format of a UUID, for example 123e4567-e89b-12d3-a456-426614174000.", what))
		return uuid.Nil, false
	}
	return id, true
}

// writeError changes an error of a use case into a problem detail.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrQuestionInUse):
		problem.Write(c, http.StatusConflict,
			"An Exam uses the Question. Remove the Question from the Exam, or delete the Exam. Then delete the Question.")
	case errors.Is(err, domain.ErrExamLocked):
		problem.Write(c, http.StatusConflict,
			"Candidates started this Exam. Only the close time can change, and you cannot delete the Exam.")
	case errors.Is(err, domain.ErrQuestionLocked):
		problem.Write(c, http.StatusConflict, "An Attempt used this Question. You cannot change the Question.")
	case errors.Is(err, domain.ErrExamLinkNotFound):
		problem.Write(c, http.StatusNotFound, "The Exam link is not valid. Check the link.")
	case errors.Is(err, domain.ErrAttemptNotFound):
		problem.Write(c, http.StatusNotFound, "The Attempt does not exist. Check the address of the page.")
	case errors.Is(err, domain.ErrAttemptClosed):
		problem.Write(c, http.StatusConflict, "The Attempt is closed. The Time limit ended, or you submitted the Attempt. You cannot save more Answers.")
	case errors.Is(err, domain.ErrAttemptAlreadySubmitted):
		problem.Write(c, http.StatusConflict, "The Attempt is already submitted. You cannot submit it again.")
	case errors.Is(err, domain.ErrInvalidAnswer):
		problem.Write(c, http.StatusUnprocessableEntity,
			"The Answer is not valid. The Question is not in this Exam, or the Option is not an Option of the Question. Select an Option of the Question.")
	case errors.Is(err, domain.ErrExamNotOpen):
		problem.Write(c, http.StatusConflict, "The Exam is not open yet. Try again after the open time.")
	case errors.Is(err, domain.ErrExamClosed):
		problem.Write(c, http.StatusConflict, "The Exam is closed. You cannot start it now.")
	case errors.Is(err, domain.ErrAttemptFinished):
		problem.Write(c, http.StatusConflict, "This Exam is already submitted for this email address. You cannot start it again.")
	case errors.Is(err, domain.ErrEmptyCandidateName):
		problem.Write(c, http.StatusUnprocessableEntity, "The name is empty. Enter your name.")
	case errors.Is(err, domain.ErrCandidateNameTooLong):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The name is too long. Use %d characters or fewer.", domain.MaxCandidateNameLength))
	case errors.Is(err, domain.ErrInvalidEmailAddress):
		problem.Write(c, http.StatusUnprocessableEntity,
			"The email address is not valid. Enter an address in the format name@example.com.")
	case errors.Is(err, domain.ErrExamNotFound):
		problem.Write(c, http.StatusNotFound, "The Exam does not exist. Check the ID.")
	case errors.Is(err, domain.ErrEmptyTitle):
		problem.Write(c, http.StatusUnprocessableEntity, "The title of the Exam is empty. Enter a title.")
	case errors.Is(err, domain.ErrTitleTooLong):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The title of the Exam is too long. Use %d characters or fewer.", domain.MaxTitleLength))
	case errors.Is(err, domain.ErrTooFewQuestions):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The Exam has too few Questions. Select %d to %d Questions.", domain.MinQuestions, domain.MaxQuestions))
	case errors.Is(err, domain.ErrTooManyQuestions):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The Exam has too many Questions. Select %d to %d Questions.", domain.MinQuestions, domain.MaxQuestions))
	case errors.Is(err, domain.ErrDuplicateQuestion):
		problem.Write(c, http.StatusUnprocessableEntity, "The Exam has the same Question 2 times. Select each Question one time.")
	case errors.Is(err, domain.ErrUnknownQuestion):
		problem.Write(c, http.StatusUnprocessableEntity, "A Question of the Exam does not exist. Select Questions from the Question bank.")
	case errors.Is(err, domain.ErrInvalidTimeLimit):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The Time limit is not valid. Enter a number of minutes from %d to %d.", domain.MinTimeLimitMinutes, domain.MaxTimeLimitMinutes))
	case errors.Is(err, domain.ErrInvalidWindow):
		problem.Write(c, http.StatusUnprocessableEntity, "The Exam window is not valid. Give an open time that is before the close time.")
	case errors.Is(err, domain.ErrQuestionNotFound):
		problem.Write(c, http.StatusNotFound, "The Question does not exist. Check the ID.")
	case errors.Is(err, domain.ErrEmptyQuestionText):
		problem.Write(c, http.StatusUnprocessableEntity, "The text of the Question is empty. Enter the text.")
	case errors.Is(err, domain.ErrQuestionTextTooLong):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The text of the Question is too long. Use %d characters or fewer.", domain.MaxQuestionTextLength))
	case errors.Is(err, domain.ErrTextNotAllowed):
		problem.Write(c, http.StatusUnprocessableEntity,
			"The text has a character that is not allowed. Remove the control character, for example a NUL character.")
	case errors.Is(err, domain.ErrTooFewOptions):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The Question has too few Options. Give %d to %d Options.", domain.MinOptions, domain.MaxOptions))
	case errors.Is(err, domain.ErrTooManyOptions):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The Question has too many Options. Give %d to %d Options.", domain.MinOptions, domain.MaxOptions))
	case errors.Is(err, domain.ErrEmptyOptionText):
		problem.Write(c, http.StatusUnprocessableEntity, "The text of an Option is empty. Enter the text of each Option.")
	case errors.Is(err, domain.ErrOptionTextTooLong):
		problem.Write(c, http.StatusUnprocessableEntity,
			fmt.Sprintf("The text of an Option is too long. Use %d characters or fewer.", domain.MaxOptionTextLength))
	case errors.Is(err, domain.ErrCorrectOptionCount):
		problem.Write(c, http.StatusUnprocessableEntity, "The Question must have exactly one correct Option. Mark one Option as correct.")
	default:
		h.log.ErrorContext(c.Request.Context(), "The application did not complete the request.",
			slog.String("method", c.Request.Method), slog.String("route", c.FullPath()), slog.Any("error", err))
		problem.WriteInternalError(c)
	}
}
