package examhttp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/httpjson"
)

type optionRequest struct {
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}

type questionRequest struct {
	Text    string          `json:"text"`
	Options []optionRequest `json:"options"`
}

func (r questionRequest) input() app.QuestionInput {
	options := make([]domain.OptionInput, len(r.Options))
	for i, option := range r.Options {
		options[i] = domain.OptionInput{Text: option.Text, Correct: option.Correct}
	}
	return app.QuestionInput{Text: r.Text, Options: options}
}

type optionResponse struct {
	ID      uuid.UUID `json:"id"`
	Text    string    `json:"text"`
	Correct bool      `json:"correct"`
}

type questionResponse struct {
	ID        uuid.UUID        `json:"id"`
	Text      string           `json:"text"`
	Options   []optionResponse `json:"options"`
	CreatedAt time.Time        `json:"createdAt"`
}

func toQuestionResponse(question domain.Question) questionResponse {
	options := make([]optionResponse, len(question.Options))
	for i, option := range question.Options {
		options[i] = optionResponse{ID: option.ID, Text: option.Text, Correct: option.Correct}
	}
	return questionResponse{ID: question.ID, Text: question.Text, Options: options, CreatedAt: question.CreatedAt}
}

func (h *Handler) postQuestion(c *gin.Context) {
	var request questionRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	question, err := h.uc.CreateQuestion.Execute(c.Request.Context(), request.input())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toQuestionResponse(question))
}

func (h *Handler) getQuestions(c *gin.Context) {
	questions, err := h.uc.ListQuestions.Execute(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response := make([]questionResponse, len(questions))
	for i, question := range questions {
		response[i] = toQuestionResponse(question)
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) getQuestionByID(c *gin.Context) {
	id, ok := parseID(c, "a Question")
	if !ok {
		return
	}
	question, err := h.uc.GetQuestion.Execute(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toQuestionResponse(question))
}

func (h *Handler) putQuestion(c *gin.Context) {
	id, ok := parseID(c, "a Question")
	if !ok {
		return
	}
	var request questionRequest
	if !httpjson.Decode(c, &request) {
		return
	}
	question, err := h.uc.UpdateQuestion.Execute(c.Request.Context(), id, request.input())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toQuestionResponse(question))
}

func (h *Handler) removeQuestion(c *gin.Context) {
	id, ok := parseID(c, "a Question")
	if !ok {
		return
	}
	if err := h.uc.DeleteQuestion.Execute(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
