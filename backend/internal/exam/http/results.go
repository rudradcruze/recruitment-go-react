package examhttp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// The response has no Attempt ID. The Attempt ID is the secret of the Candidate.
type attemptResultResponse struct {
	Name         string               `json:"name"`
	EmailAddress string               `json:"emailAddress"`
	Status       domain.AttemptStatus `json:"status"`
	Score        *scoreResponse       `json:"score"`
	StartedAt    time.Time            `json:"startedAt"`
	FinishedAt   *time.Time           `json:"finishedAt"`
}

func (h *Handler) getExamAttempts(c *gin.Context) {
	id, ok := parseID(c, "an Exam")
	if !ok {
		return
	}
	results, err := h.uc.ListExamAttempts.Execute(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response := make([]attemptResultResponse, len(results))
	for i, result := range results {
		row := attemptResultResponse{
			Name:         result.Attempt.CandidateName,
			EmailAddress: string(result.Attempt.EmailAddress),
			Status:       result.Status,
			StartedAt:    result.Attempt.StartedAt,
			FinishedAt:   result.FinishedAt,
		}
		if result.Score != nil {
			row.Score = &scoreResponse{Correct: result.Score.Correct, Total: result.Score.Total}
		}
		response[i] = row
	}
	c.JSON(http.StatusOK, response)
}
