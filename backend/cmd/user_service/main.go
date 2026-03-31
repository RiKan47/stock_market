package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"stock-market/pkg/models"

	"github.com/google/uuid"
)

type UserService struct {
	Users      map[string]*models.User
	Portfolios map[string]map[string]*models.UserPortfolio // userID -> symbol -> portfolio
	mu         sync.RWMutex
}

func NewUserService() *UserService {
	return &UserService{
		Users:      make(map[string]*models.User),
		Portfolios: make(map[string]map[string]*models.UserPortfolio),
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	us := NewUserService()

	mux := http.NewServeMux()
	mux.HandleFunc("/users/register", us.handleRegister)
	mux.HandleFunc("/users/balance", us.handleGetBalance)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	corsMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	log.Printf("User Service listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMux))
}

func (us *UserService) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	us.mu.Lock()
	defer us.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.Balance == 0 {
		user.Balance = 10000.0 // Default starting balance
	}

	us.Users[user.ID] = &user
	us.Portfolios[user.ID] = make(map[string]*models.UserPortfolio)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (us *UserService) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	us.mu.RLock()
	defer us.mu.RUnlock()

	user, ok := us.Users[userID]
	if !ok {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
