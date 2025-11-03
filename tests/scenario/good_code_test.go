package main

import (
	"testing"
)

// GOOD: Table-driven test pattern
func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected int
	}{
		{
			name:     "all high scores",
			input:    []int{60, 70, 80},
			expected: 30, // 3 * 10
		},
		{
			name:     "all medium scores",
			input:    []int{30, 40, 45},
			expected: 15, // 3 * 5
		},
		{
			name:     "all low scores",
			input:    []int{5, 10, 15},
			expected: 0,
		},
		{
			name:     "mixed scores",
			input:    []int{60, 30, 10},
			expected: 15, // 10 + 5 + 0
		},
		{
			name:     "empty input",
			input:    []int{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test will take time due to ProcessTimeout
			// In real tests, you'd want to make timeout configurable
			result := CalculateScore(tt.input)
			if result != tt.expected {
				t.Errorf("CalculateScore(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// GOOD: Table-driven test for ProcessPayment
func TestProcessPayment(t *testing.T) {
	tests := []struct {
		name      string
		amount    float64
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid payment",
			amount:    100.0,
			wantError: false,
		},
		{
			name:      "negative amount",
			amount:    -50.0,
			wantError: true,
			errorMsg:  "amount cannot be negative",
		},
		{
			name:      "exceeds maximum",
			amount:    15000.0,
			wantError: true,
			errorMsg:  "exceeds maximum limit",
		},
		{
			name:      "zero amount",
			amount:    0.0,
			wantError: false,
		},
		{
			name:      "at maximum limit",
			amount:    MaxPaymentAmount,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProcessPayment(tt.amount)

			if tt.wantError {
				if err == nil {
					t.Errorf("ProcessPayment(%f) expected error but got nil", tt.amount)
				}
			} else {
				if err != nil {
					t.Errorf("ProcessPayment(%f) unexpected error: %v", tt.amount, err)
				}
			}
		})
	}
}
