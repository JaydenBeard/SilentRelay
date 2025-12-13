package sms

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// ClickSendService handles SMS sending via ClickSend API
type ClickSendService struct {
	username string
	apiKey   string
	from     string
	client   *http.Client
	logger   *log.Logger
}

// ClickSendMessage represents the message payload for ClickSend API
type ClickSendMessage struct {
	To     string `json:"to"`
	Body   string `json:"body"`
	From   string `json:"from,omitempty"`
	Source string `json:"source,omitempty"`
}

// ClickSendRequest represents the full API request payload
type ClickSendRequest struct {
	Messages []ClickSendMessage `json:"messages"`
}

// ClickSendResponse represents the API response
type ClickSendResponse struct {
	HTTPCode     int    `json:"http_code"`
	ResponseCode string `json:"response_code"`
	ResponseMsg  string `json:"response_msg"`
	Data         struct {
		QueuedCount int `json:"queued_count"`
		Messages    []struct {
			Direction        string `json:"direction"`
			Date             int64  `json:"date"`
			To               string `json:"to"`
			Body             string `json:"body"`
			From             string `json:"from"`
			Schedule         int64  `json:"schedule"`
			MessageID        string `json:"message_id"`
			QueueCount       int    `json:"queue_count"`
			CreditsUsed      int    `json:"credits_used"`
			CreditsRemaining int    `json:"credits_remaining"`
		} `json:"messages"`
	} `json:"data"`
}

// ClickSendAccountResponse represents the account/balance API response for health checks
type ClickSendAccountResponse struct {
	HTTPCode     int    `json:"http_code"`
	ResponseCode string `json:"response_code"`
	ResponseMsg  string `json:"response_msg"`
	Data         struct {
		Balance        string `json:"balance"` // ClickSend returns balance as string
		CurrencySymbol string `json:"currency_symbol"`
		CurrencyCode   string `json:"currency_code"`
	} `json:"data"`
}

// NewClickSendService creates a new ClickSend SMS service
func NewClickSendService() (*ClickSendService, error) {
	username := os.Getenv("CLICKSEND_USERNAME")
	apiKey := os.Getenv("CLICKSEND_API_KEY")
	from := os.Getenv("CLICKSEND_FROM")

	if username == "" || apiKey == "" {
		return nil, fmt.Errorf("ClickSend credentials not configured: CLICKSEND_USERNAME and CLICKSEND_API_KEY required")
	}

	if from == "" {
		from = "SecureMsg" // Default sender name
	}

	return &ClickSendService{
		username: username,
		apiKey:   apiKey,
		from:     from,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.New(os.Stdout, "[CLICKSEND] ", log.Ldate|log.Ltime|log.LUTC),
	}, nil
}

// SendSMS sends an SMS message to the specified phone number
func (c *ClickSendService) SendSMS(to, message string) error {
	// Clean phone number - remove + prefix if present
	cleanNumber := to
	if len(cleanNumber) > 0 && cleanNumber[0] == '+' {
		cleanNumber = cleanNumber[1:]
	}

	c.logger.Printf("Sending SMS to cleaned number: %s (original: %s)", cleanNumber, to)

	// Validate phone number format (basic validation)
	if len(cleanNumber) < 10 || len(cleanNumber) > 15 {
		return fmt.Errorf("invalid phone number format: %s", cleanNumber)
	}

	// Prepare the request payload
	request := ClickSendRequest{
		Messages: []ClickSendMessage{
			{
				To:     cleanNumber,
				Body:   message,
				From:   c.from,
				Source: "sdk",
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Prevent integer overflow in bit shift - cap at 10 iterations
			safeAttempt := attempt - 1
			if safeAttempt > 10 {
				safeAttempt = 10
			}
			delay := baseDelay * time.Duration(1<<uint(safeAttempt))
			c.logger.Printf("Retrying SMS send in %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		// Create HTTP request
		req, err := http.NewRequest("POST", "https://rest.clicksend.com/v3/sms/send", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

		// Send request
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send SMS: %w", err)
			continue
		}

		// Read response
		body, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Parse response
		var response ClickSendResponse
		if err := json.Unmarshal(body, &response); err != nil {
			c.logger.Printf("Failed to parse ClickSend response: %s", string(body))
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			continue
		}

		// Check for success
		if resp.StatusCode != http.StatusOK || response.ResponseCode != "SUCCESS" {
			c.logger.Printf("ClickSend API error - Status: %d, Code: %s, Message: %s",
				resp.StatusCode, response.ResponseCode, response.ResponseMsg)
			lastErr = fmt.Errorf("SMS send failed: %s", response.ResponseMsg)
			continue
		}

		// Log success
		if len(response.Data.Messages) > 0 {
			msg := response.Data.Messages[0]
			c.logger.Printf("SMS sent successfully - ID: %s, To: %s, Credits used: %d, Remaining: %d",
				msg.MessageID, msg.To, msg.CreditsUsed, msg.CreditsRemaining)
		}

		return nil
	}

	return lastErr
}

// HealthCheck verifies that the ClickSend service is operational by checking account balance
func (c *ClickSendService) HealthCheck() error {
	c.logger.Printf("Performing ClickSend service health check")

	// Create HTTP request to account endpoint
	req, err := http.NewRequest("GET", "https://rest.clicksend.com/v3/account", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

	// Send request with shorter timeout for health checks
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ClickSend health check failed - service unreachable: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}

	// Parse response
	var accountResp ClickSendAccountResponse
	if err := json.Unmarshal(body, &accountResp); err != nil {
		c.logger.Printf("Failed to parse ClickSend account response: %s", string(body))
		return fmt.Errorf("failed to parse health check response: %w", err)
	}

	// Check for success
	if resp.StatusCode != http.StatusOK || accountResp.ResponseCode != "SUCCESS" {
		c.logger.Printf("ClickSend health check failed - Status: %d, Code: %s, Message: %s",
			resp.StatusCode, accountResp.ResponseCode, accountResp.ResponseMsg)
		return fmt.Errorf("ClickSend service health check failed: %s", accountResp.ResponseMsg)
	}

	// Log successful health check
	c.logger.Printf("ClickSend service health check passed - Balance: %s%s",
		accountResp.Data.CurrencySymbol, accountResp.Data.Balance)

	return nil
}

// SendVerificationCode sends a verification code via SMS
func (c *ClickSendService) SendVerificationCode(phoneNumber, code string) error {
	message := fmt.Sprintf("Your SilentRelay verification code is: %s\n\nThis code expires in 5 minutes.", code)
	return c.SendSMS(phoneNumber, message)
}

// getBasicAuth creates the basic authentication header value
func (c *ClickSendService) getBasicAuth() string {
	auth := c.username + ":" + c.apiKey
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
