package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Order struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Total     float64 `json:"total"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type CreateOrderRequest struct {
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Total     float64 `json:"total"`
}

var orders []Order

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "order-service",
	})
}

func ordersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(orders)
		return
	}
	if r.Method == http.MethodPost {
		var req CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		order := Order{
			ID:        strconv.Itoa(rand.Intn(100000)),
			UserID:    req.UserID,
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			Total:     req.Total,
			Status:    "pending",
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		orders = append(orders, order)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(order)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/orders", ordersHandler)
	fmt.Println("order-service running on port 3003")
	http.ListenAndServe(":3003", nil)
}
