package contentdirectory

import (
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/store"
	"log/slog"
	"net/http"
	"os"
	"path"
)

func HandleThumbnailURL(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")

	imageName := path.Base(p)
	squire := path.Base(path.Dir(p)) == "s"
	objectID := fs_utils.NameWithoutExtension(imageName)

	if objectID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	imagePath, modTime, err := store.GetThumbnail(objectID, squire)
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
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, r, imageName, modTime, file)
}
