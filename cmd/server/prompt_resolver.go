package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgtype"
	"actionfy/db"
)

var secretRegex = regexp.MustCompile(`\{\{secrets\.([a-zA-Z0-9_-]+)\}\}`)
var envVarRegex = regexp.MustCompile(`\{\{env\.([a-zA-Z0-9_-]+)\}\}`)
var webhookBodyRegex = regexp.MustCompile(`\{\{webhook\.body\.([a-zA-Z0-9_-]+)\}\}`)
var stateRegex = regexp.MustCompile(`\{\{state\.([a-zA-Z0-9._-]+)\}\}`)

func resolvePrompt(ctx context.Context, userID string, taskID pgtype.UUID, executionID string, rawPrompt string, parentTaskID pgtype.UUID, triggerPayload map[string]interface{}) (string, int, bool, error) {
	resolved := rawPrompt
	resolvedSecrets := make(map[string]string)
	secretCount := 0

	// 1. Resolve Secrets: {{secrets.NAME}}
	// Find all unique secret names first
	matches := secretRegex.FindAllStringSubmatch(resolved, -1)
	if len(matches) > 0 {
		secretNames := make([]string, 0)
		uniqueNames := make(map[string]bool)
		for _, m := range matches {
			if len(m) >= 2 && !uniqueNames[m[1]] {
				uniqueNames[m[1]] = true
				secretNames = append(secretNames, m[1])
			}
		}

		// Bulk fetch (fallback to iterative if bulk query doesn't exist, but we have GetUserSecret)
		// For elite optimization, we use iterative but cached results to avoid redundant DB hits for same key
		resolved = secretRegex.ReplaceAllStringFunc(resolved, func(match string) string {
			submatches := secretRegex.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			secretName := submatches[1]

			if val, ok := resolvedSecrets[secretName]; ok {
				return val
			}

			encryptedVal, err := queries.GetUserSecret(ctx, db.GetUserSecretParams{
				UserID: userID,
				Name:   secretName,
			})
			if err != nil {
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
	}

	// 2. Resolve Environment Variables: {{env.KEY}}
	envVars, err := queries.GetTaskWorkspaceEnvVars(ctx, taskID)
	if err == nil {
		envMap := make(map[string]string)
		for _, ev := range envVars {
			envMap[ev.Name] = ev.Value
		}

		resolved = envVarRegex.ReplaceAllStringFunc(resolved, func(match string) string {
			submatches := envVarRegex.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			key := submatches[1]
			if val, ok := envMap[key]; ok {
				return val
			}
			return match
		})
	}

	// 3. Resolve Webhook Body Variables: {{webhook.body.FIELD}}
	if triggerPayload != nil {
		resolved = webhookBodyRegex.ReplaceAllStringFunc(resolved, func(match string) string {
			submatches := webhookBodyRegex.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			key := submatches[1]
			if val, ok := triggerPayload[key]; ok {
				return fmt.Sprintf("%v", val)
			}
			return match
		})
	}

	// 4. Resolve Workflow State: {{state.VARIABLE}}
	stateBytes, err := queries.GetWorkflowState(ctx, db.GetWorkflowStateParams{
		TaskID:      taskID,
		ExecutionID: executionID,
	})
	if err == nil {
		var stateMap map[string]interface{}
		if err := json.Unmarshal(stateBytes, &stateMap); err == nil {
			resolved = stateRegex.ReplaceAllStringFunc(resolved, func(match string) string {
				submatches := stateRegex.FindStringSubmatch(match)
				if len(submatches) < 2 {
					return match
				}
				key := submatches[1]
				if val, ok := stateMap[key]; ok {
					return fmt.Sprintf("%v", val)
				}
				return match
			})
		}
	}

	chained := false
	// 5. Resolve Chaining Context
	if parentTaskID.Valid {
		parentOutput, err := queries.GetLatestTaskLogResponse(ctx, db.GetLatestTaskLogResponseParams{
			TaskID: parentTaskID,
			UserID: userID,
		})
		if err == nil && parentOutput.Valid && parentOutput.String != "" {
			// Prepend context
			resolved = fmt.Sprintf("Context from previous task:\n%s\n\nYour Prompt:\n%s", parentOutput.String, resolved)
			chained = true
		}
	}

	return resolved, secretCount, chained, nil
}
