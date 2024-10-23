package contentdirectory

import (
	"log/slog"
	"net/http"
	"path"

	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
	"github.com/szonov/godlna/internal/fs_utils"
)

func HandleStreamIconURL(w http.ResponseWriter, r *http.Request) {
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
	iconPath := path.Join(object.FullPath(), "icon.png")
	if !fs_utils.FileExists(iconPath) {
		slog.Error("There is no icon for stream", "objectID", objectID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("transferMode.dlna.org", "Interactive")
	w.Header().Set("contentFeatures.dlna.org", dlna.NewThumbContentFeatures().String())
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, iconPath)
}
