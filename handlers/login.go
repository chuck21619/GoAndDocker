package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chuck21619/MTGBackend/db"
	"github.com/chuck21619/MTGBackend/models"
	"github.com/chuck21619/MTGBackend/utils"

	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request, database *db.Database) {
	if r.Method != http.MethodPost {
		utils.WriteJSONMessage(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var u models.User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusBadRequest, "Bad request")
		return
	}

	var storedHash string
	err = database.QueryRow("SELECT password, email_verified FROM users WHERE username = $1", u.Username).Scan(&storedHash, &u.Email_verified)
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(u.Password))
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !u.Email_verified {
		utils.WriteJSONMessage(w, http.StatusNotFound, "Email has not been verified")
		return
	}

	accessToken, _, err := utils.GenerateAccessToken(u.Username)
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusInternalServerError, "Error generating access token")
		return
	}

	refreshToken, refreshExpirationTime, err := utils.GenerateRefreshToken(u.Username)
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusInternalServerError, "error generating refresh token")
		return
	}

	hashedRefreshToken := utils.HashRefreshToken(refreshToken)
	err = database.StoreRefreshToken(u.Username, hashedRefreshToken)
	if err != nil {
		utils.WriteJSONMessage(w, http.StatusInternalServerError, "Failed to save refresh token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  refreshExpirationTime,
		HttpOnly: true,
		Secure:   true,
		Path:     "/api/refresh-token",
		SameSite: http.SameSiteNoneMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": accessToken,
		"message":      "Login successful",
	})
}
