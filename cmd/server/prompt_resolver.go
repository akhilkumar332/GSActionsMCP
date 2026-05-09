package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgtype"
	"schedule-mcp/db"
)

var secretRegex = regexp.MustCompile(`\{\{secrets\.([a-zA-Z0-9_-]+)\}\}`)

func resolvePrompt(ctx context.Context, userID string, rawPrompt string, parentTaskID pgtype.UUID) (string, int, bool, error) {
	resolved := rawPrompt
	resolvedSecrets := make(map[string]string)
	secretCount := 0

	// 1. Resolve Secrets: {{secrets.NAME}}
	// Find all matches, fetch from db, decrypt, replace
	resolved = secretRegex.ReplaceAllStringFunc(resolved, func(match string) string {
		// match is {{secrets.NAME}}
		submatches := secretRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		secretName := submatches[1]

		// Check cache
		if val, ok := resolvedSecrets[secretName]; ok {
			return val
		}

		encryptedVal, err := queries.GetUserSecret(ctx, db.GetUserSecretParams{
			UserID: userID,
			Name:   secretName,
		})
		if err != nil {
			// If secret not found, leave as is or replace with error message?
			// Let's replace with empty or error for security
			val := fmt.Sprintf("[SECRET %s NOT FOUND]", secretName)
			resolvedSecrets[secretName] = val
			return val
		}

		decryptedVal, err := Decrypt(encryptedVal)
		if err != nil {
			val := "[DECRYPTION ERROR]"
			resolvedSecrets[secretName] = val
			return val
		}

		val := string(decryptedVal)
		resolvedSecrets[secretName] = val
		secretCount++
		return val
	})

	chained := false
	// 2. Resolve Chaining Context
	if parentTaskID.Valid {
		parentOutput, err := queries.GetLatestTaskLogResponse(ctx, parentTaskID)
		if err == nil && parentOutput.Valid && parentOutput.String != "" {
			// Prepend context
			resolved = fmt.Sprintf("Context from previous task:\n%s\n\nYour Prompt:\n%s", parentOutput.String, resolved)
			chained = true
		}
	}

	return resolved, secretCount, chained, nil
}
