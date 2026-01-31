package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"path"
	"runtime"
	"strings"
	"time"
)

// LogError logs an error with detailed context information
func LogError(ctx context.Context, err error, errorLogger *log.Logger) {
	if err == nil {
		return
	}

	file := "unknown"
	line := 0

	if _, f, l, ok := runtime.Caller(2); ok {
		file = path.Base(f)
		line = l
	}

	meta, ok := FromContext(ctx)

	if ok {
		ts := time.Now().Format("15:04:05")
		sep := fmt.Sprintf(
			"==============================CRITICAL[%s]==================================",
			ts,
		)

		errorLogger.Printf("[%s]", "ERROR")
		printRaw(errorLogger, sep)

		printRaw(
			errorLogger,
			fmt.Sprintf(
				`ERROR  : %v
REQ    : %s
FROM   : %s:%d
HTTP   : %s %s (%s)
UA     : %s
STACK  :
%s`,
				err,
				meta.RequestID,
				path.Base(file),
				line,
				meta.Method,
				meta.Path,
				meta.IP,
				meta.UserAgent,
				prettyStackList(3, 6),
			),
		)

		printRaw(errorLogger, "\n"+sep)
		return
	}

	errorLogger.Printf(
		"[CRITICAL] err=%v",
		err,
	)
}

// printRaw prints a message without timestamp/file info
func printRaw(l *log.Logger, s string) {
	oldFlags := l.Flags()
	l.SetFlags(0)
	l.Println(s)
	l.SetFlags(oldFlags)
}

// prettyStackList formats stack trace for readable output
func prettyStackList(skip, max int) string {
	var b strings.Builder

	for i := skip; i < skip+max; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		name := "unknown"
		if fn != nil {
			name = path.Base(fn.Name())
		}

		b.WriteString(fmt.Sprintf(
			"- %-28s %s\n",
			fmt.Sprintf("%s:%d", path.Base(file), line),
			name,
		))
	}

	return strings.TrimRight(b.String(), "\n")
}

// LogErrorLoki logs an error in JSON format suitable for Loki
func LogErrorLoki(ctx context.Context, service string, level string, err error, writer io.Writer) {
	if err == nil {
		return
	}

	meta, _ := FromContext(ctx)
	_, file, line, _ := runtime.Caller(2)

	ev := map[string]interface{}{
		"ts":         time.Now().Format(time.RFC3339),
		"level":      strings.ToUpper(level),
		"service":    service,
		"error":      err.Error(),
		"request_id": meta.RequestID,
		"http": map[string]string{
			"method": meta.Method,
			"path":   meta.Path,
			"ip":     meta.IP,
			"ua":     meta.UserAgent,
		},
		"source": map[string]interface{}{
			"file": path.Base(file),
			"line": line,
		},
		"stack": stackFrames(3, 6),
	}

	b, _ := jsonMarshal(ev)
	writer.Write(append(b, '\n'))
}

// stackFrames returns stack trace as string slice
func stackFrames(skip, max int) []string {
	var frames []string

	for i := skip; i < skip+max; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		name := "unknown"
		if fn != nil {
			name = path.Base(fn.Name())
		}

		frames = append(
			frames,
			fmt.Sprintf("%s:%d %s", path.Base(file), line, name),
		)
	}

	return frames
}