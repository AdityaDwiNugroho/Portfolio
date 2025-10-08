package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type MoneyLoginRequest struct {
	Password string `json:"password"`
}

type MoneyLoginResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	TriesLeft    int    `json:"tries_left,omitempty"`
	LockedUntil  int64  `json:"locked_until,omitempty"`
	Message      string `json:"message,omitempty"`
}

const (
	MaxTries = 5
	LockDuration = 15 * 60 // 15 minutes in seconds
)

func MoneyAuth(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	var req MoneyLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: false,
			Message: "Invalid request",
		})
		return
	}
	
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: false,
			Message: "Server error",
		})
		return
	}
	
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: false,
			Message: "Server error",
		})
		return
	}
	
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	
	// Check if locked
	lockedUntilStr, err := rdb.Get(ctx, "money_locked_until").Result()
	if err == nil && lockedUntilStr != "" {
		lockedUntil, _ := strconv.ParseInt(lockedUntilStr, 10, 64)
		now := time.Now().Unix()
		
		if now < lockedUntil {
			json.NewEncoder(w).Encode(MoneyLoginResponse{
				Success: false,
				LockedUntil: lockedUntil,
				Message: "Too many failed attempts. Please try again later.",
			})
			return
		} else {
			// Lock expired, reset tries
			rdb.Del(ctx, "money_locked_until")
			rdb.Del(ctx, "money_failed_tries")
		}
	}
	
	// Get stored password
	storedPassword, err := rdb.Get(ctx, "money_password").Result()
	if err != nil {
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: false,
			Message: "Password not configured",
		})
		return
	}
	
	// Check password
	if req.Password == storedPassword {
		// Success - reset failed tries
		rdb.Del(ctx, "money_failed_tries")
		rdb.Del(ctx, "money_locked_until")
		
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: true,
			Token: "money_access_granted",
			Message: "Login successful",
		})
		return
	}
	
	// Failed attempt
	failedTriesStr, _ := rdb.Get(ctx, "money_failed_tries").Result()
	failedTries := 0
	if failedTriesStr != "" {
		failedTries, _ = strconv.Atoi(failedTriesStr)
	}
	failedTries++
	
	rdb.Set(ctx, "money_failed_tries", strconv.Itoa(failedTries), 0)
	
	triesLeft := MaxTries - failedTries
	
	if triesLeft <= 0 {
		// Lock the account
		lockedUntil := time.Now().Unix() + LockDuration
		rdb.Set(ctx, "money_locked_until", strconv.FormatInt(lockedUntil, 10), 0)
		
		json.NewEncoder(w).Encode(MoneyLoginResponse{
			Success: false,
			TriesLeft: 0,
			LockedUntil: lockedUntil,
			Message: "Account locked due to too many failed attempts. Try again in 15 minutes.",
		})
		return
	}
	
	json.NewEncoder(w).Encode(MoneyLoginResponse{
		Success: false,
		TriesLeft: triesLeft,
		Message: "Incorrect password. " + strconv.Itoa(triesLeft) + " tries remaining.",
	})
}
