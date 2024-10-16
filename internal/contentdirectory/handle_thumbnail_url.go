package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/fs_util"
	"log/slog"
	"net/http"
	"os"
)

func HandleThumbnailURL(w http.ResponseWriter, r *http.Request) {
	profile := client.GetProfileByRequest(r)
	imageName := r.PathValue("path")
	objectID := fs_util.NameWithoutExtension(imageName)
	if objectID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	imagePath, modTime, err := backend.GetThumbnail(objectID, profile)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var file *os.File
	if file, err = os.Open(imagePath); err != nil {
		slog.Error("open thumb file", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			slog.Error("close thumb file", "err", err.Error())
		}
	}(file)

	w.Header().Set("transferMode.dlna.org", "Interactive")
	w.Header().Set("contentFeatures.dlna.org", contentThumbnailFeatures())

	// thumbnail always jpeg image
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, r, imageName, modTime, file)
}
