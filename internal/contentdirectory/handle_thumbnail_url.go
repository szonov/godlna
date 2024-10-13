package contentdirectory

import (
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/thumbnails"
	"log/slog"
	"net/http"
	"os"
)

func HandleThumbnailURL(w http.ResponseWriter, r *http.Request) {
	profile := client.GetProfileByRequest(r)
	image := r.PathValue("image")
	if image == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	info, err := thumbnails.GetImageInfo(image, profile)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var file *os.File
	if file, err = os.Open(info.Path); err != nil {
		slog.Error("open thumb file", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			slog.Error("close thumb file", "err", err.Error())
		}
	}(file)

	w.Header().Set("Content-Type", info.Mime)
	http.ServeContent(w, r, info.Name, info.Time, file)
}
