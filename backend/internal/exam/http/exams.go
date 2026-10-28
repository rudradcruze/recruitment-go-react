package examhttp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/httpjson"
)

type examRequest struct {
	Title            string      `json:"title"`
	QuestionIDs      []uuid.UUID `json:"questionIds"`
	OpensAt          time.Time   `json:"opensAt"`
	ClosesAt         time.Time   `json:"closesAt"`
	TimeLimitMinutes int         `json:"timeLimitMinutes"`
}

func (r examRequest) input() domain.ExamInput {
	return domain.ExamInput{
		Title: r.Title, QuestionIDs: r.QuestionIDs, OpensAt: r.OpensAt, ClosesAt: r.ClosesAt, TimeLimitMinutes: r.TimeLimitMinutes,
	}
}

type examResponse struct {
	ID               uuid.UUID   `json:"id"`
	Title            string      `json:"title"`
	QuestionIDs      []uuid.UUID `json:"questionIds"`
	OpensAt          time.Time   `json:"opensAt"`
	ClosesAt         time.Time   `json:"closesAt"`
	TimeLimitMinutes int         `json:"timeLimitMinutes"`
	LinkToken        string      `json:"linkToken"`
	CreatedAt        time.Time   `json:"createdAt"`
}

type examSummaryResponse struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	OpensAt          time.Time `json:"opensAt"`
	ClosesAt         time.Time `json:"closesAt"`
	TimeLimitMinutes int       `json:"timeLimitMinutes"`
	QuestionCount    int       `json:"questionCount"`
	CreatedAt        time.Time `json:"createdAt"`
}

func toExamResponse(exam domain.Exam) examResponse {
	return examResponse{
		ID: exam.ID, Title: exam.Title, QuestionIDs: exam.QuestionIDs, OpensAt: exam.OpensAt, ClosesAt: exam.ClosesAt,
		TimeLimitMinutes: exam.TimeLimitMinutes, LinkToken: exam.LinkToken, CreatedAt: exam.CreatedAt,
	}
}

func (h *Handler) postExam(c *gin.Context) {
	var request examRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	exam, err := h.uc.CreateExam.Execute(c.Request.Context(), request.input())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toExamResponse(exam))
}

func (h *Handler) getExams(c *gin.Context) {
	exams, err := h.uc.ListExams.Execute(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response := make([]examSummaryResponse, len(exams))
	for i, exam := range exams {
		response[i] = examSummaryResponse{
			ID: exam.ID, Title: exam.Title, OpensAt: exam.OpensAt, ClosesAt: exam.ClosesAt,
			TimeLimitMinutes: exam.TimeLimitMinutes, QuestionCount: len(exam.QuestionIDs), CreatedAt: exam.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) getExamByID(c *gin.Context) {
	id, ok := parseID(c, "an Exam")
	if !ok {
		return
	}
	exam, err := h.uc.GetExam.Execute(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) putExam(c *gin.Context) {
	id, ok := parseID(c, "an Exam")
	if !ok {
		return
	}
	var request examRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	exam, err := h.uc.UpdateExam.Execute(c.Request.Context(), id, request.input())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) postExamLink(c *gin.Context) {
	id, ok := parseID(c, "an Exam")
	if !ok {
		return
	}
	exam, err := h.uc.RegenerateExamLink.Execute(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) removeExam(c *gin.Context) {
	id, ok := parseID(c, "an Exam")
	if !ok {
		return
	}
	if err := h.uc.DeleteExam.Execute(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
