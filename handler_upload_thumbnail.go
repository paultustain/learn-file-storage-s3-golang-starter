package main

import (
	"fmt"
	"net/http"
	"io"
	"time"
	"path/filepath"
	"os"
	"strings"
	"mime"

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
	
	contentType := fileheader.Header["Content-Type"]
	mediaType, _, err := mime.ParseMediaType(contentType[0])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse media type", err)
		return 
	}

	if !(mediaType == "image/jpeg" || mediaType == "image/png") {
		fmt.Println(mediaType)
		respondWithError(w, http.StatusUnauthorized, "Invalid media type", err)
		return 
	}
	
	dbVideo, err := cfg.db.GetVideo(videoID) 
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Failed to get video from db", err)
		return 
	}
	
	fileID := fmt.Sprintf("%s.%s", videoID, strings.Split(contentType[0], "/")[1])
	thumbnailPath := filepath.Join(cfg.assetsRoot, fileID)

	assetFile , err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file", err)
		return 
	}
	defer assetFile.Close() 

	if _, err = io.Copy(assetFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save file", err)
		return 
	}

	
	dataURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileID)
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

