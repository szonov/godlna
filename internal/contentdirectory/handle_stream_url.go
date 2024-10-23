package contentdirectory

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"syscall"

	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
)

func HandleStreamURL(w http.ResponseWriter, r *http.Request) {
	objectID := r.PathValue("objectID")
	object := db.GetObject(objectID)

	if object == nil {
		slog.Error("Object path not found", "objectID", objectID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if object.Type != db.TypeStream {
		slog.Error("object is not stream", "objectID", objectID)
		return
	}

	if object.Meta == nil {
		slog.Error("object has no meta information", "objectID", objectID)
		return
	}

	w.Header().Set("EXT", "")
	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", dlna.NewMediaContentFeatures(object.Profile()).String())
	w.Header().Set("Content-Type", object.MimeType())

	if r.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
		return
	}

	command := object.Meta.(*db.StreamMeta).Command
	slog.Debug("Stream", "request_id", r.Header.Get("X-Request-ID"), "command", strings.Join(command, " "))

	var stream io.ReadCloser
	var err error

	if stream, err = startStreaming(command); err != nil {
		slog.Error("Start Stream: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer stopStreaming(stream)

	w.WriteHeader(http.StatusOK)

	if _, err = io.Copy(w, stream); err != nil {
		if !errors.Is(err, syscall.EPIPE) {
			slog.Error("Stream copy: " + err.Error())
		}
	}
	slog.Debug("Stream complete", "request_id", r.Header.Get("X-Request-ID"))
}

type streamErrorLogger struct {
}

func (s *streamErrorLogger) Write(p []byte) (n int, err error) {
	slog.Error("Live Stream: " + string(p))
	return len(p), nil
}

func startStreaming(command []string) (stream io.ReadCloser, err error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stderr = new(streamErrorLogger)
	if stream, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	go func() {
		_ = cmd.Wait()
	}()
	return
}

func stopStreaming(stream io.ReadCloser) {
	if err := stream.Close(); err != nil {
		slog.Error("Close Stream: " + err.Error())
	}
}
