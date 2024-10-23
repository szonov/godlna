package contentdirectory

import (
	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
	"log/slog"
	"net/http"
)

func HandleVideoURL(w http.ResponseWriter, r *http.Request) {
	objectID := r.PathValue("objectID")
	object := db.GetObject(objectID)
	if object == nil {
		slog.Error("Object path not found", "objectID", objectID)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("EXT", "")
	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", dlna.NewMediaContentFeatures().String())
	w.Header().Set("Content-Type", object.MimeType())
	http.ServeFile(w, r, object.FullPath())
}
