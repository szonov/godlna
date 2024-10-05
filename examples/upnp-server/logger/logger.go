package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func rgb(s string, r uint, g uint, b uint) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[m", r, g, b, s)
}

type MyLogHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *MyLogHandler) Handle(ctx context.Context, r slog.Record) error {

	// caller
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	caller := f.File + ":" + strconv.Itoa(f.Line)
	if len(caller) > 20 {
		caller = ".." + caller[len(caller)-20:]
	}

	// values
	values := make([]string, 0)
	r.Attrs(func(a slog.Attr) bool {
		key := rgb(a.Key, 154, 237, 254)
		val := a.Value.String()
		if a.Key == "err" {
			val = rgb(a.Value.String(), 255, 92, 87)
		}
		values = append(values, key+"="+val)
		return true
	})

	// level
	level := r.Level.String()
	message := r.Message
	switch r.Level {
	case slog.LevelDebug:
		level = rgb("DBG", 200, 200, 200)
	case slog.LevelInfo:
		level = rgb("INF", 100, 150, 200)
	case slog.LevelWarn:
		level = rgb("WRN", 243, 249, 157)
	case slog.LevelError:
		level = rgb("ERR", 255, 92, 87)
		message = rgb(message, 255, 92, 87)
	}

	if strings.HasPrefix(message, "\n") {
		message = rgb(message, 243, 249, 157)
	}

	h.l.Println(
		rgb(r.Time.Format("2006-01-02 15:04:05"), 119, 119, 119),
		level,
		rgb(caller, 87, 195, 255),
		message,
		strings.Join(values, " "),
	)

	return nil
}

func NewMyLogHandler(out io.Writer, level slog.Level) *MyLogHandler {
	h := &MyLogHandler{
		Handler: slog.NewTextHandler(out, &slog.HandlerOptions{
			AddSource: true,
			Level:     level,
		}),
		l: log.New(out, "", 0),
	}
	return h
}

func InitLogger(level ...slog.Level) {
	if len(level) == 0 {
		level = append(level, slog.LevelDebug)
	}
	logger := slog.New(NewMyLogHandler(os.Stdout, level[0]))
	slog.SetDefault(logger)
}
