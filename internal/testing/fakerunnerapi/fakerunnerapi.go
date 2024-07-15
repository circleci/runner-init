package fakerunnerapi

import (
	"context"
	"net/http"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/httpserver/ginrouter"
	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/testing/httprecorder"
	"github.com/circleci/ex/testing/httprecorder/ginrecorder"
	"github.com/gin-gonic/gin"
)

type RunnerAPI struct {
	*httprecorder.RequestRecorder
	http.Handler
	tasks []Task
}

type Task struct {
	ID           string        `json:"task_id"`
	Token        secret.String `json:"token"`
	Allocation   string        `json:"allocation"`
	UnclaimCount int           `json:"unclaim_count"`
}

func New(ctx context.Context, tasks []Task) *RunnerAPI {
	r := ginrouter.Default(ctx, "fake-runner-api")

	rec := httprecorder.New()
	r.Use(ginrecorder.Middleware(ctx, rec))

	ra := &RunnerAPI{
		RequestRecorder: rec,
		Handler:         r,
		tasks:           tasks,
	}

	r.Use(ra.authHandler)
	r.POST("/api/v2/task/event/fail", ra.failTaskHandler)
	r.POST("/api/v3/runner/unclaim", ra.unclaimHandler)

	return ra
}

func (r *RunnerAPI) failTaskHandler(c *gin.Context) {
	task := r.findTask(c.Request)

	body := struct {
		Allocation string `json:"allocation"`
		Timestamp  int64  `json:"timestamp"` // milliseconds
		Message    []byte `json:"message"`
	}{}
	err := c.BindJSON(&body)

	switch {
	case err != nil:
		c.AbortWithStatus(http.StatusBadRequest)
	case body.Allocation != task.Allocation:
		c.AbortWithStatus(http.StatusNotFound)
	default:
		c.AbortWithStatus(http.StatusOK)
	}
}

func (r *RunnerAPI) unclaimHandler(c *gin.Context) {
	task := r.findTask(c.Request)

	body := struct {
		ID    string `json:"task_id" binding:"required"`
		Token string `json:"task_token" binding:"required"`
	}{}
	err := c.BindJSON(&body)

	switch {
	case err != nil:
		c.AbortWithStatus(http.StatusBadRequest)
	case task.ID != body.ID:
		c.AbortWithStatus(http.StatusBadRequest)
	case task.UnclaimCount >= 3:
		c.AbortWithStatus(http.StatusConflict)
	default:
		c.AbortWithStatus(http.StatusOK)
	}
}

func (r *RunnerAPI) findTask(req *http.Request) *Task {
	token, _ := bearerAuth(req)
	for _, task := range r.tasks {
		if string(task.Token) == string(token) {
			return &task
		}
	}
	return nil
}

func (r *RunnerAPI) authHandler(c *gin.Context) {
	ctx := c.Request.Context()

	_, ok := bearerAuth(c.Request)
	if !ok {
		o11y.AddField(ctx, "token_authed", "no-token")
		abort(c)
		return
	}

	if r.findTask(c.Request) == nil {
		o11y.AddField(ctx, "token_authed", "invalid-token")
		abort(c)
		return
	}

	o11y.AddField(ctx, "token_authed", "success")
	c.Next()
}

func bearerAuth(r *http.Request) (token secret.String, ok bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}
	return parseBearerAuth(auth)
}

func parseBearerAuth(auth string) (token secret.String, ok bool) {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
		return token, ok
	}
	return secret.String(auth[len(prefix):]), true
}

func abort(c *gin.Context) {
	c.AbortWithStatus(http.StatusNotFound)
}