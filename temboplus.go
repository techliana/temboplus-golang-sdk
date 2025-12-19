package temboplus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents the TemboPlus API client
type Client struct {
	baseURL    string
	accountID  string
	secretKey  string
	httpClient *http.Client
}

// ClientConfig holds configuration for the TemboPlus client
type ClientConfig struct {
	Environmen Environment   // "sandbox" or "production"
	AccountID  string        // Your account ID (x-account-id)
	SecretKey  string        // Your secret key (x-secret-key)
	Timeout    time.Duration // Default: 30 seconds
}
type Environment string

const (
	Sandbox    Environment = "sandbox"
	Production Environment = "production"
)

// NewClient creates a new TemboPlus client
func NewClient(config ClientConfig) *Client {
	baseUrl := DefaultBaseURLSandbox
	if config.Environmen == Production {
		baseUrl = DefaultBaseURLProduction
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL:   baseUrl,
		accountID: config.AccountID,
		secretKey: config.SecretKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (e Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("TemboPlus API Error [%s]: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("TemboPlus API Error: %s", e.StatusCode)
}

// generateRequestID creates a unique request ID for the x-request-id header
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// makeRequest handles HTTP requests to the TemboPlus API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, payload interface{}) (*MobileMoneyCollectionResponse, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-account-id", c.accountID)
	req.Header.Set("x-secret-key", c.secretKey)
	req.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// If HTTP status is not OK, try to unmarshal API error wrapper
	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var response MobileMoneyCollectionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error status codes
	if response.StatusCode == StatusPaymentRejected || response.StatusCode == StatusGenericError {
		return &response, Error{
			StatusCode: response.StatusCode,
			Message:    "Request failed",
		}
	}

	return &response, nil
}

// CollectFromMobileMoney sends a USSD push request to collect money from a mobile subscriber
func (c *Client) CollectFromMobileMoney(ctx context.Context, req MobileMoneyCollectionRequest) (*MobileMoneyCollectionResponse, error) {
	// Validate required fields
	if err := c.validateMobileMoneyRequest(req); err != nil {
		return nil, err
	}

	response, err := c.makeRequest(ctx, http.MethodPost, EndpointCollection, req)
	if err != nil {
		return response, err
	}

	return response, nil
}

// validateMobileMoneyRequest validates the mobile money collection request
func (c *Client) validateMobileMoneyRequest(req MobileMoneyCollectionRequest) error {
	if req.MSISDN == "" {
		return fmt.Errorf("MSISDN is required")
	}
	if req.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Narration == "" {
		return fmt.Errorf("narration is required")
	}
	if req.TransactionRef == "" {
		return fmt.Errorf("transactionRef is required")
	}
	if req.TransactionDate == "" {
		return fmt.Errorf("transactionDate is required")
	}
	if req.CallbackURL == "" {
		return fmt.Errorf("callbackUrl is required")
	}

	// Validate channel
	if !isValidChannel(req.Channel) {
		return fmt.Errorf("invalid channel: %s. Supported channels: %v", req.Channel, GetSupportedChannels())
	}

	return nil
}

// ValidateWebhook validates and parses an incoming webhook payload
func (c *Client) ValidateWebhook(payload []byte) (*WebhookPayload, error) {
	var webhook WebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Basic validation
	if webhook.TransactionRef == "" || webhook.TransactionID == "" {
		return nil, fmt.Errorf("invalid webhook payload: missing required fields")
	}

	return &webhook, nil
}

// GetCollectionBalance retrieves the balance of the collection account
func (c *Client) GetCollectionBalance(ctx context.Context) (*CollectionBalanceResponse, error) {
	url := c.baseURL + EndpointWalletCollectionBalance

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-account-id", c.accountID)
	req.Header.Set("x-secret-key", c.secretKey)
	req.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result CollectionBalanceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetMainBalance retrieves the balance of the main account
func (c *Client) GetMainBalance(ctx context.Context) (*CollectionBalanceResponse, error) {
	url := c.baseURL + EndpointWalletMainBalance

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-account-id", c.accountID)
	req.Header.Set("x-secret-key", c.secretKey)
	req.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result CollectionBalanceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetCollectionStatement retrieves a list of statement entries for the collection account within a date range
func (c *Client) GetCollectionStatement(ctx context.Context, reqBody CollectionStatementRequest) ([]CollectionStatementEntry, error) {
	// Marshal payload
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + EndpointWalletCollectionStatement
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-account-id", c.accountID)
	req.Header.Set("x-secret-key", c.secretKey)
	req.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var entries []CollectionStatementEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return entries, nil
}

// GetMainStatement retrieves a list of statement entries for the main account within a date range
func (c *Client) GetMainStatement(ctx context.Context, reqBody CollectionStatementRequest) ([]CollectionStatementEntry, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + EndpointWalletMainStatement
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-account-id", c.accountID)
	req.Header.Set("x-secret-key", c.secretKey)
	req.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var entries []CollectionStatementEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return entries, nil
}

// GetCollectionStatus checks the payment status using transactionRef and/or transactionId
func (c *Client) GetCollectionStatus(ctx context.Context, req PaymentStatusRequest) (*MobileMoneyCollectionResponse, error) {
	// Basic validation: require at least one identifier
	if req.TransactionRef == "" && req.TransactionID == "" {
		return nil, fmt.Errorf("either transactionRef or transactionId is required")
	}

	return c.makeRequest(ctx, http.MethodPost, EndpointCollectionStatus, req)
}

// GetPaymentStatus checks the status of a payment (wallet-to-mobile, wallet-to-bank, utilities)
func (c *Client) GetPaymentStatus(ctx context.Context, req PaymentStatusRequest) (*MobileMoneyCollectionResponse, error) {
	if req.TransactionRef == "" && req.TransactionID == "" {
		return nil, fmt.Errorf("either transactionRef or transactionId is required")
	}
	return c.makeRequest(ctx, http.MethodPost, EndpointPaymentStatus, req)
}

// Constants for status codes
const (
	StatusPendingACK      = "PENDING_ACK"
	StatusPaymentAccepted = "PAYMENT_ACCEPTED"
	StatusPaymentRejected = "PAYMENT_REJECTED"
	StatusGenericError    = "GENERIC_ERROR"
)

// Constants for supported channels
const (
	ChannelTZTigoC2B    = "TZ-TIGO-C2B"
	ChannelTZHalotelC2B = "TZ-HALOTEL-C2B"
	ChannelTZAirtelC2B  = "TZ-AIRTEL-C2B"
	ServiceTZTigoB2C    = "TZ-TIGO-B2C"
	ServiceTZAirtelB2C  = "TZ-AIRTEL-B2C"
	ServiceTZBankB2C    = "TZ-BANK-B2C"
)

// Helper functions

// GetSupportedChannels returns a list of supported MNO channels
func GetSupportedChannels() []string {
	return []string{
		ChannelTZTigoC2B,
		ChannelTZAirtelC2B,
		ChannelTZHalotelC2B,
	}
}

// isValidChannel checks if the provided channel is supported
func isValidChannel(channel string) bool {
	supportedChannels := GetSupportedChannels()
	for _, c := range supportedChannels {
		if c == channel {
			return true
		}
	}
	return false
}

// GetSupportedServices returns supported wallet-to-mobile service codes
func GetSupportedServices() []string {
	return []string{
		ServiceTZTigoB2C,
		ServiceTZAirtelB2C,
		ServiceTZBankB2C,
	}
}

// isValidService checks if provided service code is supported
func isValidService(service string) bool {
	for _, s := range GetSupportedServices() {
		if s == service {
			return true
		}
	}
	return false
}

// FormatMSISDN formats a phone number to the required MSISDN format (255XXX123456)
func FormatMSISDN(phoneNumber string) string {
	// Remove any leading + or 0
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		phoneNumber = phoneNumber[1:]
	}
	if len(phoneNumber) > 0 && phoneNumber[0] == '0' {
		phoneNumber = phoneNumber[1:]
	}

	// If it doesn't start with country code, assume Tanzania (255)
	if len(phoneNumber) == 9 && (phoneNumber[0] == '6' || phoneNumber[0] == '7') {
		phoneNumber = "255" + phoneNumber
	}

	return phoneNumber
}

// GenerateTransactionRef generates a unique transaction reference
func GenerateTransactionRef(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d", prefix, timestamp)
}

// FormatTransactionDate formats a time.Time to the required transaction date format
func FormatTransactionDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// IsSuccessfulWebhook checks if a webhook indicates a successful payment
func IsSuccessfulWebhook(webhook *WebhookPayload) bool {
	return webhook.StatusCode == StatusPaymentAccepted
}

// IsFailedWebhook checks if a webhook indicates a failed payment
func IsFailedWebhook(webhook *WebhookPayload) bool {
	return webhook.StatusCode == StatusPaymentRejected || webhook.StatusCode == StatusGenericError
}

// GetChannelProvider extracts the provider name from a channel
func GetChannelProvider(channel string) string {
	switch channel {
	case ChannelTZTigoC2B:
		return "Tigo"
	case ChannelTZAirtelC2B:
		return "Airtel"
	default:
		return "Unknown"
	}
}

// ValidateMSISDN performs basic validation on MSISDN format
func ValidateMSISDN(msisdn string) error {
	if len(msisdn) < 10 || len(msisdn) > 15 {
		return fmt.Errorf("invalid MSISDN length: %s", msisdn)
	}

	// Should start with country code (e.g., 255 for Tanzania)
	if len(msisdn) >= 3 && msisdn[:3] != "255" {
		return fmt.Errorf("MSISDN should start with country code 255 for Tanzania: %s", msisdn)
	}

	return nil
}

// BuildCollectionRequest is a helper function to build a properly formatted collection request
func BuildCollectionRequest(phoneNumber string, channel string, amount float64, description string, callbackURL string) MobileMoneyCollectionRequest {
	now := time.Now()

	return MobileMoneyCollectionRequest{
		MSISDN:          FormatMSISDN(phoneNumber),
		Channel:         channel,
		Amount:          amount,
		Narration:       description,
		TransactionRef:  GenerateTransactionRef("TXN"),
		TransactionDate: FormatTransactionDate(now),
		CallbackURL:     callbackURL,
	}
}

// PayWalletToBank is a convenience wrapper for bank payouts (TZ-BANK-B2C)
// Note: The API uses the same endpoint as wallet-to-mobile; msisdn should be in the format <BIC>:<ACCOUNT NUMBER>
func (c *Client) PayWalletToBank(ctx context.Context, req WalletToMobileRequest) (*MobileMoneyCollectionResponse, error) {
	if req.ServiceCode == "" {
		req.ServiceCode = ServiceTZBankB2C
	}
	if req.ServiceCode != ServiceTZBankB2C {
		return nil, fmt.Errorf("serviceCode must be %s for bank payouts", ServiceTZBankB2C)
	}
	return c.PayWalletToMobile(ctx, req)
}

// PayWalletToMobile initiates a transfer from a wallet to a mobile subscriber
func (c *Client) PayWalletToMobile(ctx context.Context, req WalletToMobileRequest) (*MobileMoneyCollectionResponse, error) {
	// Validate inputs
	if err := c.validateWalletToMobileRequest(req); err != nil {
		return nil, err
	}

	// Reuse the common request helper; response shape matches MobileMoneyCollectionResponse
	return c.makeRequest(ctx, http.MethodPost, EndpointPaymentWalletToMobile, req)
}

func (c *Client) validateWalletToMobileRequest(req WalletToMobileRequest) error {
	if req.CountryCode == "" {
		return fmt.Errorf("countryCode is required")
	}
	if req.CountryCode != "TZ" {
		return fmt.Errorf("unsupported countryCode: %s", req.CountryCode)
	}
	if req.AccountNo == "" {
		return fmt.Errorf("accountNo is required")
	}
	if req.ServiceCode == "" {
		return fmt.Errorf("serviceCode is required")
	}
	if !isValidService(req.ServiceCode) {
		return fmt.Errorf("invalid serviceCode: %s. Supported services: %v", req.ServiceCode, GetSupportedServices())
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.MSISDN == "" {
		return fmt.Errorf("msisdn is required")
	}
	if req.Narration == "" {
		return fmt.Errorf("narration is required")
	}
	if req.CurrencyCode == "" {
		return fmt.Errorf("currencyCode is required")
	}
	if req.CurrencyCode != "TZS" {
		return fmt.Errorf("unsupported currencyCode: %s", req.CurrencyCode)
	}
	if req.RecipientNames == "" {
		return fmt.Errorf("recipientNames is required")
	}
	if req.TransactionRef == "" {
		return fmt.Errorf("transactionRef is required")
	}
	if req.TransactionDate == "" {
		return fmt.Errorf("transactionDate is required")
	}
	if req.CallbackURL == "" {
		return fmt.Errorf("callbackUrl is required")
	}
	return nil
}
