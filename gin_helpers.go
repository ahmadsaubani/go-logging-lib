package logging

import (
	"github.com/gin-gonic/gin"
)

// MarkErrorLogged marks that an error has been manually logged
func MarkErrorLogged(c *gin.Context) {
	c.Set("error_logged_manually", true)
}

// IsErrorLogged checks if an error has been manually logged
func IsErrorLogged(c *gin.Context) bool {
	logged, exists := c.Get("error_logged_manually")
	return exists && logged.(bool)
}

// SetLoggedError stores the error in context for Loki logging
func SetLoggedError(c *gin.Context, err error) {
	c.Set("logged_error", err)
}

// LogErrorWithMark logs an error and marks it as logged to prevent duplication
func (l *Logger) LogErrorWithMark(c *gin.Context, err error) {
	l.Error(c.Request.Context(), err)
	SetLoggedError(c, err)
	MarkErrorLogged(c)
}