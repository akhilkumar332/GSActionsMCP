package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// RegisterUser hashes password, generates API key, and inserts user into DB
func RegisterUser(ctx context.Context, email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKeyBytes := make([]byte, 16)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}
	apiKey := hex.EncodeToString(apiKeyBytes)

	var user User
	err = dbPool.QueryRow(ctx, 
		"INSERT INTO users (email, password_hash, api_key) VALUES ($1, $2, $3) RETURNING id, email, api_key, role, tier, created_at",
		email, string(hashedPassword), apiKey,
	).Scan(&user.ID, &user.Email, &user.APIKey, &user.Role, &user.Tier, &user.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &user, nil
}

// LoginUser verifies password and creates a session
func LoginUser(ctx context.Context, email, password string) (string, error) {
	var userID, passwordHash string
	err := dbPool.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE email = $1", email).Scan(&userID, &passwordHash)
	if err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	var sessionID string
	expiresAt := time.Now().Add(24 * time.Hour)
	err = dbPool.QueryRow(ctx, 
		"INSERT INTO web_sessions (user_id, expires_at) VALUES ($1, $2) RETURNING id",
		userID, expiresAt,
	).Scan(&sessionID)

	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}
