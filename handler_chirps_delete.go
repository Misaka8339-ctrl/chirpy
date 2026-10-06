package main

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/Misaka8339-ctrl/chirpy/internal/auth"
	"github.com/Misaka8339-ctrl/chirpy/internal/database"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	// 1. 验证登录身份
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 2. 解析路径中的 Chirp ID
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	// 3. 查询 Chirp，区分不存在和无权删除
	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirp")
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You can only delete your own chirps")
		return
	}

	// 4. 删除时再次限定作者
	rowsAffected, err := cfg.db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     chirpID,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete chirp")
		return
	}

	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	// 5. 成功，无响应正文
	w.WriteHeader(http.StatusNoContent)
}
