package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func sendFailureEmail(ctx context.Context, userID string, taskID string, taskName string) {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		log.Printf("SENDGRID_API_KEY not set. Skipping email for task %s (user %s).", taskID, userID)
		return
	}

	email, err := queries.GetUserEmail(ctx, userID)
	if err != nil || email.String == "" {
		log.Printf("No valid email found for user %s. Skipping failure email.", userID)
		return
	}

	from := mail.NewEmail("Scheduled Actions Server", "noreply@yourservice.com")
	subject := fmt.Sprintf("Action Required: Task '%s' Failed", taskName)
	to := mail.NewEmail("User", email.String)
	plainTextContent := fmt.Sprintf("Your scheduled action '%s' (ID: %s) has failed 3 times and is now in an error state. Please review its configuration.", taskName, taskID)
	htmlContent := fmt.Sprintf("<strong>Your scheduled action '%s'</strong> has failed 3 times and is now in an error state. Please log in to your dashboard to review its configuration.", taskName)

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(message)
	if err != nil {
		log.Printf("SendGrid error: %v", err)
	} else {
		log.Printf("Failure email sent to %s. Status Code: %d", email.String, response.StatusCode)
	}
}
