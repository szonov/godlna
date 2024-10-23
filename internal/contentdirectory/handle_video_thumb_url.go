package contentdirectory

import (
	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
	"log/slog"
	"net/http"
)

func HandleVideoThumbURL(w http.ResponseWriter, r *http.Request) {
	var objectID, imagePath string
	var err error
	if objectID = r.PathValue("objectID"); objectID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if imagePath, err = db.GetVideoThumb(objectID); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("transferMode.dlna.org", "Interactive")
	w.Header().Set("contentFeatures.dlna.org", dlna.NewThumbContentFeatures().String())
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, imagePath)
}
