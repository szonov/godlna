package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"log/slog"
	"net/http"
	"os"
)

func HandleVideoURL(w http.ResponseWriter, r *http.Request) {
	video := r.PathValue("video")
	if video == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	objectId := backend.NameWithoutExt(video)
	videoPath := backend.GetObjectPath(objectId)

	if videoPath == "" {
		slog.Error("Video path not found", "objectID", objectId)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var err error

	var statInfo os.FileInfo
	statInfo, err = os.Stat(videoPath)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var file *os.File
	if file, err = os.Open(videoPath); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if r.Header.Get("getcontentFeatures.dlna.org") == "1" {

		w.Header().Set("EXT", "")
		w.Header().Set("realTimeInfo.dlna.org", "DLNA.ORG_TLAG=*")
		w.Header().Set("transferMode.dlna.org", "Streaming")
		w.Header().Set("contentFeatures.dlna.org", "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000")
	}

	//w.Header().Set("Content-Type", "video/x-matroska")
	w.Header().Set("Content-Type", "video/x-msvideo")
	http.ServeContent(w, r, video, statInfo.ModTime(), file)
}
