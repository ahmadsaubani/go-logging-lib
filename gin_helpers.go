package logging

import (
	"github.com/gin-gonic/gin"
)

func MarkErrorLogged(c *gin.Context) {
	c.Set("error_logged_manually", true)
}

func IsErrorLogged(c *gin.Context) bool {
	logged, exists := c.Get("error_logged_manually")
	return exists && logged.(bool)
}

func SetLoggedError(c *gin.Context, err error) {
	c.Set("logged_error", err)
}

/**
 * LogErrorWithMark logs an error and marks it as logged to prevent duplication.
 * Use this when manually handling errors to avoid double logging in middleware.
 *
 * @param c Gin context
 * @param err Error to log
 */
func (l *Logger) LogErrorWithMark(c *gin.Context, err error) {
	l.Error(c.Request.Context(), err)
	SetLoggedError(c, err)
	MarkErrorLogged(c)
}