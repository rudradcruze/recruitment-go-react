package examhttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/httpjson"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/problem"
)

type examLinkResponse struct {
	Title            string    `json:"title"`
	OpensAt          time.Time `json:"opensAt"`
	ClosesAt         time.Time `json:"closesAt"`
	TimeLimitMinutes int       `json:"timeLimitMinutes"`
	QuestionCount    int       `json:"questionCount"`
}

type startAttemptRequest struct {
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
}

// The types below have no field for the correct Option. A response to a Candidate cannot say which Option is correct.
type candidateOptionResponse struct {
	ID   uuid.UUID `json:"id"`
	Text string    `json:"text"`
}

type candidateQuestionResponse struct {
	ID      uuid.UUID                 `json:"id"`
	Text    string                    `json:"text"`
	Options []candidateOptionResponse `json:"options"`
}

type answerResponse struct {
	QuestionID uuid.UUID `json:"questionId"`
	OptionID   uuid.UUID `json:"optionId"`
}

type openAttemptResponse struct {
	ID        uuid.UUID                   `json:"id"`
	Status    domain.AttemptStatus        `json:"status"`
	ExamTitle string                      `json:"examTitle"`
	ExpiresAt time.Time                   `json:"expiresAt"`
	Questions []candidateQuestionResponse `json:"questions"`
	Answers   []answerResponse            `json:"answers"`
}

type finishedAttemptResponse struct {
	ID     uuid.UUID            `json:"id"`
	Status domain.AttemptStatus `json:"status"`
}

func toAttemptResponse(view app.AttemptView) any {
	if view.Status != domain.StatusOpen {
		return finishedAttemptResponse{ID: view.Attempt.ID, Status: view.Status}
	}
	questions := make([]candidateQuestionResponse, len(view.Questions))
	for i, question := range view.Questions {
		options := make([]candidateOptionResponse, len(question.Options))
		for j, option := range question.Options {
			options[j] = candidateOptionResponse{ID: option.ID, Text: option.Text}
		}
		questions[i] = candidateQuestionResponse{ID: question.ID, Text: question.Text, Options: options}
	}
	answers := make([]answerResponse, len(view.Answers))
	for i, answer := range view.Answers {
		answers[i] = answerResponse{QuestionID: answer.QuestionID, OptionID: answer.OptionID}
	}
	return openAttemptResponse{
		ID: view.Attempt.ID, Status: view.Status, ExamTitle: view.Exam.Title, ExpiresAt: view.Attempt.ExpiresAt,
		Questions: questions, Answers: answers,
	}
}

func (h *Handler) getExamLink(c *gin.Context) {
	exam, err := h.uc.GetExamLink.Execute(c.Request.Context(), c.Param("token"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, examLinkResponse{
		Title: exam.Title, OpensAt: exam.OpensAt, ClosesAt: exam.ClosesAt,
		TimeLimitMinutes: exam.TimeLimitMinutes, QuestionCount: len(exam.QuestionIDs),
	})
}

func (h *Handler) postAttempt(c *gin.Context) {
	var request startAttemptRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	view, created, err := h.uc.StartAttempt.Execute(c.Request.Context(), app.StartAttemptInput{
		Token: c.Param("token"), Name: request.Name, EmailAddress: request.EmailAddress,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, toAttemptResponse(view))
}

// parseUUID reads a UUID in the canonical form with dashes. uuid.Parse also accepts other forms, for example with braces.
// It writes the problem detail and returns false when the value is not valid.
func parseUUID(c *gin.Context, raw, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil || id.String() != strings.ToLower(raw) {
		problem.Write(c, http.StatusBadRequest, fmt.Sprintf(
			"The ID is not valid. Use the ID of %s, in the format of a UUID, for example 123e4567-e89b-12d3-a456-426614174000.", what))
		return uuid.Nil, false
	}
	return id, true
}

type saveAnswerRequest struct {
	OptionID uuid.UUID `json:"optionId"`
}

type scoreResponse struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

func (h *Handler) putAnswer(c *gin.Context) {
	attemptID, ok := parseUUID(c, c.Param("id"), "an Attempt")
	if !ok {
		return
	}
	questionID, ok := parseUUID(c, c.Param("questionId"), "a Question")
	if !ok {
		return
	}
	var request saveAnswerRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	err := h.uc.SaveAnswer.Execute(c.Request.Context(), attemptID, domain.Answer{QuestionID: questionID, OptionID: request.OptionID})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) postSubmission(c *gin.Context) {
	attemptID, ok := parseUUID(c, c.Param("id"), "an Attempt")
	if !ok {
		return
	}
	score, err := h.uc.SubmitAttempt.Execute(c.Request.Context(), attemptID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, scoreResponse{Correct: score.Correct, Total: score.Total})
}

func (h *Handler) getAttemptByID(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "an Attempt")
	if !ok {
		return
	}
	view, err := h.uc.GetAttempt.Execute(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAttemptResponse(view))
}
