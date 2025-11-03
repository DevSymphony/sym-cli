package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GOOD: Using environment variable for API key
var APIKey = os.Getenv("OPENAI_API_KEY")

// Named constants instead of magic numbers
const (
	MaxPaymentAmount     = 10000.0
	MinPaymentAmount     = 0.0
	HighScoreThreshold   = 50
	MediumScoreThreshold = 20
	HighScorePoints      = 10
	MediumScorePoints    = 5
	ProcessTimeout       = 300 * time.Second
)

// ProcessPayment processes a payment transaction for the given amount.
// It returns an error if the amount is negative or exceeds the maximum limit.
// GOOD: Has godoc comment for exported function
func ProcessPayment(amount float64) error {
	// GOOD: Returning error instead of panic
	if amount < MinPaymentAmount {
		return fmt.Errorf("amount cannot be negative: %.2f", amount)
	}

	// GOOD: Using named constant
	if amount > MaxPaymentAmount {
		return fmt.Errorf("amount exceeds maximum limit of %.2f", MaxPaymentAmount)
	}

	return nil
}

// UserRepository defines the interface for user data access
// GOOD: Repository pattern for database access
type UserRepository interface {
	Create(username, email string) error
	FindByEmail(email string) (*User, error)
}

type User struct {
	ID       int
	Username string
	Email    string
}

// PostgresUserRepository implements UserRepository
type PostgresUserRepository struct {
	db *sql.DB
}

// Create creates a new user in the database
// GOOD: Using prepared statement to prevent SQL injection
func (r *PostgresUserRepository) Create(username, email string) error {
	query := "INSERT INTO users (username, email) VALUES ($1, $2)"
	_, err := r.db.Exec(query, username, email)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindByEmail finds a user by email address
func (r *PostgresUserRepository) FindByEmail(email string) (*User, error) {
	query := "SELECT id, username, email FROM users WHERE email = $1"
	var user User

	// GOOD: Checking error return
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &user, nil
}

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userRepo UserRepository
}

// NewUserHandler creates a new UserHandler with injected dependencies
// GOOD: Dependency injection through constructor
func NewUserHandler(userRepo UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// HandleCreateUser handles user creation requests
// GOOD: Using repository instead of direct database access
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")

	// GOOD: Checking error return and handling properly
	err := h.userRepo.Create(username, email)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User created successfully"))
}

// ProcessOrders processes a list of orders
// GOOD: Reduced nesting by extracting helper functions
func ProcessOrders(orders []Order) {
	for _, order := range orders {
		if order.Valid {
			processOrderItems(order.Items)
		}
	}
}

func processOrderItems(items []Item) {
	for _, item := range items {
		if item.Available {
			processItemOptions(item.Options)
		}
	}
}

func processItemOptions(options []Option) {
	for _, option := range options {
		if option.Selected {
			fmt.Printf("Processing option: %s\n", option.Name)
		}
	}
}

// CalculateScore calculates a score based on input data
// GOOD: Using named constants instead of magic numbers
func CalculateScore(data []int) int {
	score := 0
	for _, val := range data {
		if val > HighScoreThreshold {
			score += HighScorePoints
		} else if val > MediumScoreThreshold {
			score += MediumScorePoints
		}
	}

	time.Sleep(ProcessTimeout)

	return score
}

// Helper types
type Order struct {
	Valid bool
	Items []Item
}

type Item struct {
	Available bool
	Options   []Option
}

type Option struct {
	Selected bool
	Name     string
}
