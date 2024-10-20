package contentdirectory

import (
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/store"
	"log/slog"
	"net/http"
	"os"
	"path"
)

func HandleVideoURL(w http.ResponseWriter, r *http.Request) {
	video := path.Base(r.PathValue("path"))
	if video == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	objectId := fs_utils.NameWithoutExtension(video)
	object := store.GetObject(objectId)

	if object == nil {
		slog.Error("Object path not found", "objectID", objectId)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var err error

	var statInfo os.FileInfo
	statInfo, err = os.Stat(object.FullPath())
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var file *os.File
	if file, err = os.Open(object.FullPath()); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	w.Header().Set("EXT", "")
	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", contentVideoFeatures())
	w.Header().Set("Content-Type", object.MimeType())
	http.ServeContent(w, r, video, statInfo.ModTime(), file)
}
