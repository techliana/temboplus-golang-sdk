package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/techliana/temboplus"
	// Replace with actual import path
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}
}
func main() {
	// Initialize the client with your credentials
	client := temboplus.NewClient(temboplus.ClientConfig{
		Environmen: temboplus.Production,
		AccountID:  os.Getenv("ACCOUNT_ID"), // Your x-account-id
		SecretKey:  os.Getenv("SECRET_KEY"), // Your x-secret-key
		Timeout:    30 * time.Second,
	})

	// Example 1: Simple mobile money collection
	mobileMoneyCollectionExample(client)

	return

	// Example 0a: Get main account balance
	getMainBalanceExample(client)

	// Example 0b: Get collection account balance
	getCollectionBalanceExample(client)

	// Example 0b: Get collection account statement
	getCollectionStatementExample(client)

	// Example 0c: Get main account statement
	getMainStatementExample(client)

	// Example 2: Collection with different channels
	multiChannelExample(client)

	// Example 3: Webhook handling
	setupWebhookServer(client)

	// Example 4: Advanced collection scenarios
	advancedCollectionExamples(client)

	// Example 5: Query a payment status (wallet/mobile/bank)
	getPaymentStatusExample(client)

	// Example 6: Wallet to Mobile payment
	walletToMobileExample(client)

	// Example 7: Wallet to Bank payment
	walletToBankExample(client)
}

func getCollectionBalanceExample(client *temboplus.Client) {
	fmt.Println("=== Collection Balance Example ===")
	ctx := context.Background()
	balance, err := client.GetCollectionBalance(ctx)
	if err != nil {
		log.Printf("Error retrieving collection balance: %v", err)
		return
	}
	fmt.Printf("Available: %s\n", formatCurrency(balance.AvailableBalance))
	fmt.Printf("Current:   %s\n", formatCurrency(balance.CurrentBalance))
	fmt.Printf("Account:   %s (%s)\n", balance.AccountNo, balance.AccountStatus)
	fmt.Printf("Name:      %s\n\n", balance.AccountName)
}

func getMainBalanceExample(client *temboplus.Client) {
	fmt.Println("=== Main Balance Example ===")
	ctx := context.Background()
	balance, err := client.GetMainBalance(ctx)
	if err != nil {
		log.Printf("Error retrieving main balance: %v", err)
		return
	}
	fmt.Printf("Available: %s\n", formatCurrency(balance.AvailableBalance))
	fmt.Printf("Current:   %s\n", formatCurrency(balance.CurrentBalance))
	fmt.Printf("Account:   %s (%s)\n", balance.AccountNo, balance.AccountStatus)
	fmt.Printf("Name:      %s\n\n", balance.AccountName)
}

func getCollectionStatementExample(client *temboplus.Client) {
	fmt.Println("=== Collection Statement Example ===")
	ctx := context.Background()
	req := temboplus.CollectionStatementRequest{
		StartDate: "2023-01-01",
		EndDate:   "2023-01-31",
		// WalletID: "optional-wallet-id-if-required",
	}
	entries, err := client.GetCollectionStatement(ctx, req)
	if err != nil {
		log.Printf("Error retrieving collection statement: %v", err)
		return
	}
	fmt.Printf("Entries: %d\n", len(entries))
	// Print first few entries
	max := 3
	if len(entries) < max {
		max = len(entries)
	}
	for i := 0; i < max; i++ {
		e := entries[i]
		var credited, debited float64
		if e.AmountCredited.Value != nil {
			credited = *e.AmountCredited.Value
		}
		if e.AmountDebited.Value != nil {
			debited = *e.AmountDebited.Value
		}
		fmt.Printf("%d) %s %s %s CR:%.2f DR:%.2f BAL:%.2f\n", i+1, e.TxnDate, e.AccountNo, e.Narration, credited, debited, e.Balance)
	}
	fmt.Println()
}

func getMainStatementExample(client *temboplus.Client) {
	fmt.Println("=== Main Statement Example ===")
	ctx := context.Background()
	req := temboplus.CollectionStatementRequest{
		StartDate: "2023-01-01",
		EndDate:   "2023-01-31",
	}
	entries, err := client.GetMainStatement(ctx, req)
	if err != nil {
		log.Printf("Error retrieving main statement: %v", err)
		return
	}
	fmt.Printf("Entries: %d\n", len(entries))
	max := 3
	if len(entries) < max {
		max = len(entries)
	}
	for i := 0; i < max; i++ {
		e := entries[i]
		var credited, debited float64
		if e.AmountCredited.Value != nil {
			credited = *e.AmountCredited.Value
		}
		if e.AmountDebited.Value != nil {
			debited = *e.AmountDebited.Value
		}
		fmt.Printf("%d) %s %s %s CR:%.2f DR:%.2f BAL:%.2f\n", i+1, e.TxnDate, e.AccountNo, e.Narration, credited, debited, e.Balance)
	}
	fmt.Println()
}

func getPaymentStatusExample(client *temboplus.Client) {
	fmt.Println("=== Payment Status Example ===")
	ctx := context.Background()
	req := temboplus.PaymentStatusRequest{
		TransactionRef: "Hyu8373HmsI",
		TransactionID:  "X50jcLDcU",
	}
	resp, err := client.GetPaymentStatus(ctx, req)
	if err != nil {
		log.Printf("Error getting payment status: %v", err)
		return
	}
	fmt.Printf("Status: %s, TxnID: %s, Ref: %s\n\n", resp.StatusCode, resp.TransactionID, resp.TransactionRef)
}

func walletToMobileExample(client *temboplus.Client) {
	fmt.Println("=== Wallet to Mobile Example (B2C) ===")
	ctx := context.Background()
	req := temboplus.WalletToMobileRequest{
		CountryCode:     "TZ",
		AccountNo:       "8000837333", // replace with your main/customer wallet account no
		ServiceCode:     temboplus.ServiceTZTigoB2C,
		Amount:          2500,
		MSISDN:          temboplus.FormatMSISDN("0715123456"),
		Narration:       "Payout - Order #123",
		CurrencyCode:    "TZS",
		RecipientNames:  "John Doe",
		TransactionRef:  temboplus.GenerateTransactionRef("PAYOUT"),
		TransactionDate: temboplus.FormatTransactionDate(time.Now()),
		CallbackURL:     "https://your-app.com/webhooks/temboplus",
	}
	resp, err := client.PayWalletToMobile(ctx, req)
	if err != nil {
		log.Printf("Wallet to Mobile error: %v", err)
		return
	}
	fmt.Printf("Submitted. Status: %s, TxnID: %s, Ref: %s\n\n", resp.StatusCode, resp.TransactionID, resp.TransactionRef)
}

func walletToBankExample(client *temboplus.Client) {
	fmt.Println("=== Wallet to Bank Example (B2C) ===")
	ctx := context.Background()
	req := temboplus.WalletToMobileRequest{
		CountryCode: "TZ",
		AccountNo:   "8000837333", // replace with your main/customer wallet account no
		// For bank payouts, serviceCode must be TZ-BANK-B2C
		ServiceCode: temboplus.ServiceTZBankB2C,
		Amount:      1000,
		// msisdn must be in the format <BIC>:<ACCOUNT NUMBER>
		MSISDN:          "CORUTZTZ:0150078564433",
		Narration:       "Salary advance to John Doe",
		CurrencyCode:    "TZS",
		RecipientNames:  "JOHN DOE",
		TransactionRef:  temboplus.GenerateTransactionRef("BANKPAY"),
		TransactionDate: temboplus.FormatTransactionDate(time.Now()),
		CallbackURL:     "http://example.com/webhook",
	}
	resp, err := client.PayWalletToBank(ctx, req)
	if err != nil {
		log.Printf("Wallet to Bank error: %v", err)
		return
	}
	fmt.Printf("Submitted. Status: %s, TxnID: %s, Ref: %s\n\n", resp.StatusCode, resp.TransactionID, resp.TransactionRef)
}

func mobileMoneyCollectionExample(client *temboplus.Client) {
	fmt.Println("=== Mobile Money Collection Example ===")

	ctx := context.Background()

	// Create a collection request for Tigo Tanzania
	request := temboplus.MobileMoneyCollectionRequest{
		MSISDN:          temboplus.FormatMSISDN("0715123456"),                        // Will format to 255715123456
		Channel:         temboplus.ChannelTZTigoC2B,                                  // Tigo Tanzania
		Amount:          10000.0,                                                     // Amount in TZS
		Narration:       "Payment for online purchase - Order #123",                  // Description
		TransactionRef:  temboplus.GenerateTransactionRef("ORDER"),                   // Your unique reference
		TransactionDate: temboplus.FormatTransactionDate(time.Now()),                 // Current timestamp
		CallbackURL:     "https://webhook.site/31483592-710d-4d80-a533-c2b15ceb284a", // Your webhook endpoint
	}

	response, err := client.CollectFromMobileMoney(ctx, request)
	if err != nil {
		log.Printf("Error initiating collection: %v", err)
		return
	}

	fmt.Printf("Collection Request Submitted Successfully!\n")
	fmt.Printf("Status Code: %s\n", response.StatusCode)
	fmt.Printf("Transaction Reference: %s\n", response.TransactionRef)
	fmt.Printf("Transaction ID: %s\n", response.TransactionID)

	switch response.StatusCode {
	case temboplus.StatusPendingACK:
		fmt.Printf("✅ USSD push sent to customer. Waiting for payment confirmation.\n")
	case temboplus.StatusPaymentRejected:
		fmt.Printf("❌ Payment request was rejected.\n")
	case temboplus.StatusGenericError:
		fmt.Printf("⚠️ An error occurred while processing the request.\n")
	}
	fmt.Println()
}

func multiChannelExample(client *temboplus.Client) {
	fmt.Println("=== Multi-Channel Collection Example ===")

	ctx := context.Background()

	// Example with different MNO providers
	examples := []struct {
		provider    string
		channel     string
		phoneNumber string
		description string
	}{
		{
			provider:    "Tigo",
			channel:     temboplus.ChannelTZTigoC2B,
			phoneNumber: "0715123456",
			description: "Tigo customer payment",
		},
		{
			provider:    "Airtel",
			channel:     temboplus.ChannelTZAirtelC2B,
			phoneNumber: "0785123456",
			description: "Airtel customer payment",
		},
	}

	for _, example := range examples {
		fmt.Printf("Processing %s payment...\n", example.provider)

		request := temboplus.BuildCollectionRequest(
			example.phoneNumber,
			example.channel,
			5000.0, // 5,000 TZS
			example.description,
			"https://your-app.com/webhooks/temboplus",
		)

		response, err := client.CollectFromMobileMoney(ctx, request)
		if err != nil {
			log.Printf("Error with %s collection: %v", example.provider, err)
			continue
		}

		fmt.Printf("  %s Collection - Status: %s, TxnID: %s\n",
			example.provider, response.StatusCode, response.TransactionID)
	}
	fmt.Println()
}

func setupWebhookServer(client *temboplus.Client) {
	fmt.Println("=== Setting up Webhook Server ===")

	// Create webhook handler
	http.HandleFunc("/webhooks/temboplus", func(w http.ResponseWriter, r *http.Request) {
		handleTemboWebhook(client, w, r)
	})

	// Create a simple status endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Webhook server is running"))
	})

	fmt.Println("Webhook server configured:")
	fmt.Println("  - Webhook endpoint: /webhooks/temboplus")
	fmt.Println("  - Health check: /health")
	fmt.Println("  - Start server with: go run main.go")
	fmt.Println("  - Then run: http.ListenAndServe(\":8080\", nil)")
	fmt.Println()
}

func handleTemboWebhook(client *temboplus.Client, w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the raw webhook for debugging
	log.Printf("Received webhook: %s", string(body))

	// Validate and parse the webhook
	webhook, err := client.ValidateWebhook(body)
	if err != nil {
		log.Printf("Invalid webhook payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process the webhook
	processWebhookPayload(webhook)

	// Respond with 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func processWebhookPayload(webhook *temboplus.WebhookPayload) {
	fmt.Printf("\n🔔 Webhook Received:\n")
	fmt.Printf("  Transaction Reference: %s\n", webhook.TransactionRef)
	fmt.Printf("  Transaction ID: %s\n", webhook.TransactionID)
	fmt.Printf("  Status: %s\n", webhook.StatusCode)

	switch webhook.StatusCode {
	case temboplus.StatusPaymentAccepted:
		handleSuccessfulPayment(webhook)
	case temboplus.StatusPaymentRejected:
		handleRejectedPayment(webhook)
	case temboplus.StatusGenericError:
		handlePaymentError(webhook)
	default:
		log.Printf("Unknown webhook status: %s", webhook.StatusCode)
	}
}

func handleSuccessfulPayment(webhook *temboplus.WebhookPayload) {
	fmt.Printf("✅ Payment Successful!\n")
	fmt.Printf("  Processing successful payment for transaction: %s\n", webhook.TransactionRef)

	// Your business logic here:
	// - Update order status to paid
	// - Send confirmation email/SMS to customer
	// - Trigger fulfillment process
	// - Update your database
	// - Send thank you message

	// Example:
	// updateOrderStatus(webhook.TransactionRef, "paid")
	// sendConfirmationEmail(webhook.TransactionRef)
	// triggerFulfillment(webhook.TransactionRef)
}

func handleRejectedPayment(webhook *temboplus.WebhookPayload) {
	fmt.Printf("❌ Payment Rejected!\n")
	fmt.Printf("  Payment was rejected for transaction: %s\n", webhook.TransactionRef)

	// Your business logic here:
	// - Update order status to payment_failed
	// - Notify customer about failed payment
	// - Optionally retry with different amount/method
	// - Log for analysis

	// Example:
	// updateOrderStatus(webhook.TransactionRef, "payment_failed")
	// notifyCustomerPaymentFailed(webhook.TransactionRef)
}

func handlePaymentError(webhook *temboplus.WebhookPayload) {
	fmt.Printf("⚠️ Payment Error!\n")
	fmt.Printf("  Error occurred for transaction: %s\n", webhook.TransactionRef)

	// Your business logic here:
	// - Log error for investigation
	// - Update order status to error
	// - Notify support team
	// - Possibly retry later

	// Example:
	// logPaymentError(webhook.TransactionRef, "GENERIC_ERROR")
	// notifySupportTeam(webhook.TransactionRef)
}

func advancedCollectionExamples(client *temboplus.Client) {
	fmt.Println("=== Advanced Collection Examples ===")

	// Example 1: Collection with validation
	validateAndCollect(client)

	// Example 2: Batch collection processing
	batchCollectionExample(client)

	// Example 3: Collection with retry logic
	collectionWithRetry(client)
}

func validateAndCollect(client *temboplus.Client) {
	fmt.Println("--- Validation Example ---")

	phoneNumber := "0715123456"
	channel := temboplus.ChannelTZTigoC2B
	amount := 15000.0

	// Validate inputs before making API call
	formattedMSISDN := temboplus.FormatMSISDN(phoneNumber)
	if err := temboplus.ValidateMSISDN(formattedMSISDN); err != nil {
		log.Printf("Invalid MSISDN: %v", err)
		return
	}

	// Check if channel is supported
	supportedChannels := temboplus.GetSupportedChannels()
	fmt.Printf("Supported channels: %v\n", supportedChannels)

	ctx := context.Background()
	request := temboplus.BuildCollectionRequest(
		phoneNumber,
		channel,
		amount,
		"Validated payment collection",
		"https://your-app.com/webhooks/temboplus",
	)

	response, err := client.CollectFromMobileMoney(ctx, request)
	if err != nil {
		log.Printf("Collection failed: %v", err)
		return
	}

	fmt.Printf("Validated collection initiated: %s\n", response.TransactionID)
}

func batchCollectionExample(client *temboplus.Client) {
	fmt.Println("--- Batch Collection Example ---")

	ctx := context.Background()

	// Simulate multiple customers to collect from
	customers := []struct {
		phone   string
		channel string
		amount  float64
		desc    string
	}{
		{"0715111111", temboplus.ChannelTZTigoC2B, 5000, "Customer A payment"},
		{"0785222222", temboplus.ChannelTZAirtelC2B, 7500, "Customer B payment"},
		{"0715333333", temboplus.ChannelTZTigoC2B, 3000, "Customer C payment"},
	}

	results := make([]string, 0, len(customers))

	for i, customer := range customers {
		fmt.Printf("Processing customer %d/%d...\n", i+1, len(customers))

		request := temboplus.BuildCollectionRequest(
			customer.phone,
			customer.channel,
			customer.amount,
			customer.desc,
			"https://your-app.com/webhooks/temboplus",
		)

		response, err := client.CollectFromMobileMoney(ctx, request)
		if err != nil {
			log.Printf("Failed to collect from %s: %v", customer.phone, err)
			results = append(results, fmt.Sprintf("FAILED: %s", customer.phone))
			continue
		}

		results = append(results, fmt.Sprintf("SUCCESS: %s -> %s", customer.phone, response.TransactionID))

		// Add delay between requests to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Batch collection results:\n")
	for _, result := range results {
		fmt.Printf("  %s\n", result)
	}
}

func collectionWithRetry(client *temboplus.Client) {
	fmt.Println("--- Collection with Retry Logic ---")

	ctx := context.Background()
	maxRetries := 3
	retryDelay := 5 * time.Second

	request := temboplus.BuildCollectionRequest(
		"0715123456",
		temboplus.ChannelTZTigoC2B,
		8000.0,
		"Payment with retry logic",
		"https://your-app.com/webhooks/temboplus",
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Collection attempt %d/%d...\n", attempt, maxRetries)

		// Update transaction reference for each retry
		request.TransactionRef = temboplus.GenerateTransactionRef(fmt.Sprintf("RETRY_%d", attempt))
		request.TransactionDate = temboplus.FormatTransactionDate(time.Now())

		response, err := client.CollectFromMobileMoney(ctx, request)
		if err != nil {
			log.Printf("Attempt %d failed: %v", attempt, err)

			if attempt < maxRetries {
				fmt.Printf("Retrying in %v...\n", retryDelay)
				time.Sleep(retryDelay)
				continue
			} else {
				fmt.Printf("All retry attempts exhausted\n")
				break
			}
		}

		if response.StatusCode == temboplus.StatusPendingACK {
			fmt.Printf("✅ Collection successful on attempt %d: %s\n", attempt, response.TransactionID)
			break
		} else {
			fmt.Printf("❌ Collection failed with status: %s\n", response.StatusCode)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}
		}
	}
}

// Utility functions for real-world usage

func formatCurrency(amount float64) string {
	return fmt.Sprintf("TZS %.2f", amount)
}

func logTransaction(transactionRef, transactionID, status string) {
	timestamp := time.Now().Format(time.RFC3339)
	log.Printf("[%s] Transaction %s (%s): %s", timestamp, transactionRef, transactionID, status)
}

// Example of how to start the webhook server
func startWebhookServer() {
	fmt.Println("Starting webhook server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
