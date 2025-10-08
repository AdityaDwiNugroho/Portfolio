package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Transaction struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"` // "income" or "expense"
	Category    string  `json:"category"`
	Date        string  `json:"date"`
	Visibility  string  `json:"visibility"` // "public" or "private"
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

func MoneyTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Check authentication
	authHeader := r.Header.Get("Authorization")
	if authHeader != "Bearer money_access_granted" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}
	
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Server error",
		})
		return
	}
	
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Server error",
		})
		return
	}
	
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	
	switch r.Method {
	case http.MethodGet:
		handleGetTransactions(w, r, rdb, ctx)
	case http.MethodPost:
		handleCreateTransaction(w, r, rdb, ctx)
	case http.MethodPut:
		handleUpdateTransaction(w, r, rdb, ctx)
	case http.MethodDelete:
		handleDeleteTransaction(w, r, rdb, ctx)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleGetTransactions(w http.ResponseWriter, r *http.Request, rdb *redis.Client, ctx context.Context) {
	data, err := rdb.Get(ctx, "money_transactions").Result()
	
	var transactions []Transaction
	if err == nil && data != "" {
		json.Unmarshal([]byte(data), &transactions)
	} else {
		transactions = []Transaction{}
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"transactions": transactions,
	})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, rdb *redis.Client, ctx context.Context) {
	var transaction Transaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}
	
	// Generate ID if not provided
	if transaction.ID == "" {
		transaction.ID = time.Now().Format("20060102150405") + "_" + transaction.Type
	}
	transaction.CreatedAt = time.Now().Format(time.RFC3339)
	
	// Get existing transactions
	data, _ := rdb.Get(ctx, "money_transactions").Result()
	var transactions []Transaction
	if data != "" {
		json.Unmarshal([]byte(data), &transactions)
	}
	
	// Add new transaction
	transactions = append(transactions, transaction)
	
	// Save back to Redis
	jsonData, _ := json.Marshal(transactions)
	rdb.Set(ctx, "money_transactions", string(jsonData), 0)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"transaction": transaction,
		"message": "Transaction created",
	})
}

func handleUpdateTransaction(w http.ResponseWriter, r *http.Request, rdb *redis.Client, ctx context.Context) {
	var updatedTransaction Transaction
	if err := json.NewDecoder(r.Body).Decode(&updatedTransaction); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}
	
	// Get existing transactions
	data, err := rdb.Get(ctx, "money_transactions").Result()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No transactions found",
		})
		return
	}
	
	var transactions []Transaction
	json.Unmarshal([]byte(data), &transactions)
	
	// Find and update transaction
	found := false
	for i, t := range transactions {
		if t.ID == updatedTransaction.ID {
			// Preserve CreatedAt
			updatedTransaction.CreatedAt = t.CreatedAt
			transactions[i] = updatedTransaction
			found = true
			break
		}
	}
	
	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Transaction not found",
		})
		return
	}
	
	// Save back to Redis
	jsonData, _ := json.Marshal(transactions)
	rdb.Set(ctx, "money_transactions", string(jsonData), 0)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"transaction": updatedTransaction,
		"message": "Transaction updated",
	})
}

func handleDeleteTransaction(w http.ResponseWriter, r *http.Request, rdb *redis.Client, ctx context.Context) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Transaction ID required",
		})
		return
	}
	
	// Get existing transactions
	data, err := rdb.Get(ctx, "money_transactions").Result()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No transactions found",
		})
		return
	}
	
	var transactions []Transaction
	json.Unmarshal([]byte(data), &transactions)
	
	// Filter out the transaction to delete
	var newTransactions []Transaction
	found := false
	for _, t := range transactions {
		if t.ID != id {
			newTransactions = append(newTransactions, t)
		} else {
			found = true
		}
	}
	
	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Transaction not found",
		})
		return
	}
	
	// Save back to Redis
	jsonData, _ := json.Marshal(newTransactions)
	rdb.Set(ctx, "money_transactions", string(jsonData), 0)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Transaction deleted",
	})
}
