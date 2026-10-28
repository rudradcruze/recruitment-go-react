// Package adminhttp is the HTTP adapter of the Admin context.
package adminhttp

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/httpjson"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/problem"
)

const sessionCookie = "session"

// Handler serves the routes of the Admin context.
type Handler struct {
	login        *app.Login
	logout       *app.Logout
	currentAdmin *app.CurrentAdmin
	log          *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(login *app.Login, logout *app.Logout, currentAdmin *app.CurrentAdmin, log *slog.Logger) *Handler {
	return &Handler{login: login, logout: logout, currentAdmin: currentAdmin, log: log}
}

// Routes adds the routes of the context to the group of the API.
func (h *Handler) Routes(api *gin.RouterGroup) {
	session := api.Group("/admin/session")
	session.POST("", h.logIn)
	session.GET("", h.RequireSession, h.get)
	session.DELETE("", h.RequireSession, h.logOut)
}

type loginRequest struct {
	EmailAddress string `json:"emailAddress"`
	Password     string `json:"password"`
}

type adminResponse struct {
	ID           uuid.UUID `json:"id"`
	EmailAddress string    `json:"emailAddress"`
}

func (h *Handler) logIn(c *gin.Context) {
	var request loginRequest
	if !httpjson.Decode(c, &request) {
		return
	}

	token, err := h.login.Execute(c.Request.Context(), app.LoginInput{
		EmailAddress: request.EmailAddress,
		Password:     request.Password,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	setSessionCookie(c, token, int(app.SessionLifetime.Seconds()))
	c.Status(http.StatusNoContent)
}

func (h *Handler) get(c *gin.Context) {
	admin := c.MustGet(adminKey).(domain.Admin)
	c.JSON(http.StatusOK, adminResponse{ID: admin.ID, EmailAddress: string(admin.EmailAddress)})
}

func (h *Handler) logOut(c *gin.Context) {
	token, _ := c.Cookie(sessionCookie)
	if err := h.logout.Execute(c.Request.Context(), token); err != nil {
		h.writeError(c, err)
		return
	}
	setSessionCookie(c, "", -1)
	c.Status(http.StatusNoContent)
}

// shortcut: the cookie has no Secure attribute, because the stack has no TLS. Add it when a TLS proxy serves the web service.
func setSessionCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, token, maxAge, "/", "", false, true)
}

const adminKey = "admin"

// RequireSession is the guard of the routes that only an Admin can use. It answers 401 without a valid session.
// Other contexts use it as a Gin handler. They do not import this package.
func (h *Handler) RequireSession(c *gin.Context) {
	token, err := c.Cookie(sessionCookie)
	if err != nil || token == "" {
		h.writeError(c, domain.ErrNotLoggedIn)
		return
	}
	admin, err := h.currentAdmin.Execute(c.Request.Context(), token)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Set(adminKey, admin)
	c.Next()
}

// writeError changes an error of a use case into a problem detail.
func (h *Handler) writeError(c *gin.Context, err error) {
	var blocked *app.LoginBlockedError
	switch {
	case errors.As(err, &blocked):
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(blocked.RetryAfter.Seconds()))))
		problem.Write(c, http.StatusTooManyRequests,
			"There were too many failed logins for this email address. Wait, then log in again.")
	case errors.Is(err, domain.ErrInvalidCredentials):
		problem.Write(c, http.StatusUnauthorized, "The email address or the password is not correct. Enter them again.")
	case errors.Is(err, domain.ErrNotLoggedIn):
		problem.Write(c, http.StatusUnauthorized, "You are not logged in. Log in, then try again.")
	default:
		h.log.ErrorContext(c.Request.Context(), "The application did not complete the request.",
			slog.String("method", c.Request.Method), slog.String("route", c.FullPath()), slog.Any("error", err))
		problem.WriteInternalError(c)
	}
}
