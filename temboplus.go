package temboplus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents the TemboPlus API client
type Client struct {
	baseURL    string
	accountID  string
	secretKey  string
	httpClient *http.Client
	IsDebug    bool
}

// ClientConfig holds configuration for the TemboPlus client
type ClientConfig struct {
	Environmen Environment   // "sandbox" or "production"
	AccountID  string        // Your account ID (x-account-id)
	SecretKey  string        // Your secret key (x-secret-key)
	Timeout    time.Duration // Default: 30 seconds
	Debug      bool          // When true, logs formatted request and response bodies
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
		IsDebug:   config.Debug,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (e Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API Error [%s]: %s", e.StatusCode, e.Message)
	}
	if e.Details != "" {
		return fmt.Sprintf("API Error [%s]: %s %s", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("API Error: %s", e.StatusCode)
}

// generateRequestID creates a unique request ID for the x-request-id header
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// logDebug prints a labelled, pretty-printed JSON payload when IsDebug is true.
func (c *Client) logDebug(label string, data []byte) {
	if !c.IsDebug {
		return
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		fmt.Printf("[DEBUG] %s: %s\n", label, string(data))
		return
	}
	fmt.Printf("[DEBUG] %s:\n%s\n", label, buf.String())
}

// makeRequest handles HTTP requests to the TemboPlus API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, payload any) (*MobileMoneyCollectionResponse, error) {
	var body io.Reader
	var jsonData []byte
	if payload != nil {
		var err error
		jsonData, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		c.logDebug(">> Request "+method+" "+endpoint, jsonData)
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
	c.logDebug("<< Response "+endpoint, respBody)

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
	c.logDebug(">> Request POST "+EndpointWalletCollectionBalance, []byte("{}"))

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
	c.logDebug("<< Response "+EndpointWalletCollectionBalance, body)

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
	c.logDebug(">> Request POST "+EndpointWalletMainBalance, []byte("{}"))

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
	c.logDebug("<< Response "+EndpointWalletMainBalance, body)

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
	c.logDebug(">> Request POST "+EndpointWalletCollectionStatement, payload)
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
	c.logDebug("<< Response "+EndpointWalletCollectionStatement, body)

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
	c.logDebug(">> Request POST "+EndpointWalletMainStatement, payload)
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
	c.logDebug("<< Response "+EndpointWalletMainStatement, body)

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

// Constants for supported channels (C2B) and services (B2C)
const (
	// Tanzania C2B channels (mobile money collection)
	ChannelTZTigoC2B    = "TZ-TIGO-C2B"    // Mixx by Yas (formerly Tigo Pesa) - Honora Tanzania
	ChannelTZHalotelC2B = "TZ-HALOTEL-C2B" // HaloPesa - Viettel Tanzania (Halotel)
	ChannelTZAirtelC2B  = "TZ-AIRTEL-C2B"  // Airtel Money - Airtel Tanzania
	ChannelTZVodacomC2B = "TZ-VODACOM-C2B" // M-Pesa - Vodacom Tanzania

	// Tanzania B2C services (wallet-to-mobile payouts)
	ServiceTZAirtelB2C  = "TZ-AIRTEL-B2C"  // Airtel Money - Airtel Tanzania
	ServiceTZVodacomB2C = "TZ-VODACOM-B2C" // M-Pesa - Vodacom Tanzania
	ServiceTZTigoB2C    = "TZ-TIGO-B2C"    // Mixx by Yas (formerly Tigo Pesa) - Honora Tanzania
	ServiceTZHalotelB2C = "TZ-HALOTEL-B2C" // HaloPesa - Viettel Tanzania (Halotel)
	ServiceTZBankB2C    = "TZ-BANK-B2C"    // Bank payout

	// Kenya B2C services (wallet-to-mobile payouts)
	ServiceKESafaricomB2C = "KE-SAFARICOM-B2C" // M-Pesa - Safaricom Kenya
)

// Helper functions

// GetSupportedChannels returns a list of supported MNO channels
func GetSupportedChannels() []string {
	return []string{
		ChannelTZTigoC2B,
		ChannelTZAirtelC2B,
		ChannelTZHalotelC2B,
		ChannelTZVodacomC2B,
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
		ServiceTZAirtelB2C,
		ServiceTZVodacomB2C,
		ServiceTZTigoB2C,
		ServiceTZHalotelB2C,
		ServiceTZBankB2C,
		ServiceKESafaricomB2C,
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
	// remove spaces, dashes, and parentheses
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
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
		return "Mixx by Yas (Tigo)"
	case ChannelTZAirtelC2B:
		return "Airtel"
	case ChannelTZHalotelC2B:
		return "HaloPesa"
	case ChannelTZVodacomC2B:
		return "M-Pesa (Vodacom)"
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
func BuildCollectionRequest(phoneNumber string, channel string, amount int, description string, callbackURL string) MobileMoneyCollectionRequest {
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

// ListWallets retrieves all wallets associated with the account
func (c *Client) ListWallets(ctx context.Context) ([]Wallet, error) {
	url := c.baseURL + EndpointWalletList
	c.logDebug(">> Request GET "+EndpointWalletList, []byte("{}"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	c.logDebug("<< Response "+EndpointWalletList, body)

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var wallets []Wallet
	if err := json.Unmarshal(body, &wallets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return wallets, nil
}

// GetWalletBalance retrieves the balance of a specific wallet by account number
func (c *Client) GetWalletBalance(ctx context.Context, accountNo string) (*CollectionBalanceResponse, error) {
	url := c.baseURL + EndpointWalletBalance

	reqBody := WalletBalanceRequest{
		AccountNo: accountNo,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	c.logDebug(">> Request POST "+EndpointWalletBalance, jsonBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
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
	c.logDebug("<< Response "+EndpointWalletBalance, body)

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var balance CollectionBalanceResponse
	if err := json.Unmarshal(body, &balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &balance, nil
}

// TransferToDisbursement moves funds from one wallet to another (e.g. main wallet → disbursement wallet)
func (c *Client) TransferToDisbursement(ctx context.Context, req WalletTransferRequest) (*WalletTransferResponse, error) {
	if req.FromAccountNo == "" {
		return nil, fmt.Errorf("fromAccountNo is required")
	}
	if req.ToAccountNo == "" {
		return nil, fmt.Errorf("toAccountNo is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	if req.Narration == "" {
		return nil, fmt.Errorf("narration is required")
	}
	if req.TransactionRef == "" {
		return nil, fmt.Errorf("transactionRef is required")
	}
	if req.TransactionDate == "" {
		return nil, fmt.Errorf("transactionDate is required")
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	c.logDebug(">> Request POST "+EndpointWalletTransfer, jsonData)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+EndpointWalletTransfer, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-account-id", c.accountID)
	httpReq.Header.Set("x-secret-key", c.secretKey)
	httpReq.Header.Set("x-request-id", generateRequestID())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	c.logDebug("<< Response "+EndpointWalletTransfer, body)

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusCode != 0 {
			return nil, apiErr
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result WalletTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
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
