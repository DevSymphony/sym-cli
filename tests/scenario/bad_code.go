package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// VIOLATION 1: Hardcoded API key (security rule)
const APIKey = "sk-1234567890abcdefghijklmnopqrstuvwxyz"

// VIOLATION 9: Missing godoc comment for exported function
func ProcessPayment(amount float64) error {
	// VIOLATION 4: Using panic in production code
	if amount < 0 {
		panic("negative amount not allowed")
	}

	// VIOLATION 6: Magic number (should be a named constant)
	if amount > 10000 {
		return fmt.Errorf("amount exceeds limit")
	}

	return nil
}

// VIOLATION 3: Direct database access in HTTP handler (architecture rule)
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	// VIOLATION 8: Creating dependency inside function
	db, err := sql.Open("postgres", "user=postgres password=secret123 dbname=mydb")
	if err != nil {
		// VIOLATION 4: Using panic
		panic(err)
	}
	defer db.Close()

	username := r.FormValue("username")
	email := r.FormValue("email")

	// VIOLATION 2: SQL injection vulnerability (concatenating user input)
	query := "INSERT INTO users (username, email) VALUES ('" + username + "', '" + email + "')"

	// VIOLATION 5: Not checking error return
	db.Exec(query)

	w.Write([]byte("User created"))
}

// VIOLATION 7: Too many nested control structures (>3 levels)
func ProcessOrders(orders []Order) {
	for _, order := range orders {
		if order.Valid {
			for _, item := range order.Items {
				if item.Available {
					for _, option := range item.Options {
						if option.Selected {
							// Too deep nesting!
							fmt.Printf("Processing option: %s\n", option.Name)
						}
					}
				}
			}
		}
	}
}

// VIOLATION 6: Magic numbers everywhere
func CalculateScore(data []int) int {
	score := 0
	for _, val := range data {
		if val > 50 { // Magic number
			score += 10 // Magic number
		} else if val > 20 { // Magic number
			score += 5 // Magic number
		}
	}

	// More magic numbers
	time.Sleep(300 * time.Second)

	return score
}

// Helper types for examples
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

// VIOLATION 10: Not using table-driven tests (this would be in test file)
// func TestCalculateScorePositive(t *testing.T) { ... }
// func TestCalculateScoreNegative(t *testing.T) { ... }
// func TestCalculateScoreZero(t *testing.T) { ... }
// Should be one TestCalculateScore with table-driven approach
