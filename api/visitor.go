package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
)

func formatNumber(num int64) string {
	if num >= 1000000 {
		millions := float64(num) / 1000000.0
		if millions == float64(int64(millions)) {
			return fmt.Sprintf("%.0fM", millions)
		}
		return fmt.Sprintf("%.1fM", millions)
	}
	if num >= 1000 {
		thousands := float64(num) / 1000.0
		if thousands == float64(int64(thousands)) {
			return fmt.Sprintf("%.0fk", thousands)
		}
		return fmt.Sprintf("%.1fk", thousands)
	}
	return fmt.Sprintf("%d", num)
}

func Visitor(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/html")
	
	redisURL := os.Getenv("REDIS_URL")
	var count int64 = 1
	
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			rdb := redis.NewClient(opt)
			count, _ = rdb.Incr(ctx, "visitor_count").Result()
			rdb.Close()
		}
	}
	
	formattedCount := formatNumber(count)
	
	response := fmt.Sprintf(`
		<div class="stat-number">%s</div>
		<div class="stat-label">Visitors</div>
	`, formattedCount)
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
