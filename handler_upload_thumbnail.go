package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse multifile", err)
		return
	}

	file, fileheader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get thumbnail", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(fileheader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse media type", err)
		return
	}

	if !(mediaType == "image/jpeg" || mediaType == "image/png") {
		fmt.Println(mediaType)
		respondWithError(w, http.StatusUnauthorized, "Invalid media type", err)
		return
	}

	fileID := getAssetPath(videoID, mediaType)
	thumbnailPath := filepath.Join(cfg.assetsRoot, fileID)

	assetFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file", err)
		return
	}
	defer assetFile.Close()

	if _, err = io.Copy(assetFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save file", err)
		return
	}

	dbVideo, err := cfg.db.GetVideo(videoID)
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Failed to get video from db", err)
		return
	}

	dataURL := cfg.getAssetURL(thumbnailPath)
	updatedVideo := dbVideo
	updatedVideo.UpdatedAt = time.Now()
	updatedVideo.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(updatedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedVideo)
}
