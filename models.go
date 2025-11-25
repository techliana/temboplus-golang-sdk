package temboplus

import (
    "encoding/json"
    "fmt"
)

// MobileMoneyCollectionRequest represents a mobile money collection request
type MobileMoneyCollectionRequest struct {
	MSISDN          string  `json:"msisdn"`          // Phone number in format 255XXX123456
	Channel         string  `json:"channel"`         // MNO channel (TZ-TIGO-C2B, TZ-AIRTEL-C2B)
	Amount          float64 `json:"amount"`          // Amount to collect
	Narration       string  `json:"narration"`       // Description/narration
	TransactionRef  string  `json:"transactionRef"`  // Your system reference
	TransactionDate string  `json:"transactionDate"` // Date in format YYYY-MM-DD HH:mm:ss
	CallbackURL     string  `json:"callbackUrl"`     // Webhook callback URL
}

// MobileMoneyCollectionResponse represents the API response
type MobileMoneyCollectionResponse struct {
	StatusCode     string `json:"statusCode"`     // PENDING_ACK, PAYMENT_REJECTED, GENERIC_ERROR
	TransactionRef string `json:"transactionRef"` // Your system reference
	TransactionID  string `json:"transactionId"`  // TemboPlus transaction ID
}

// WebhookPayload represents the webhook callback payload
type WebhookPayload struct {
    StatusCode     string `json:"statusCode"`     // PAYMENT_ACCEPTED, PAYMENT_REJECTED, GENERIC_ERROR
    TransactionRef string `json:"transactionRef"` // Your system reference
    TransactionID  string `json:"transactionId"`  // TemboPlus transaction ID
}

// Error represents an API error
type Error struct {
    StatusCode string `json:"statusCode"`
    Message    string `json:"message,omitempty"`
    Details    string `json:"details,omitempty"`
}

// APIError represents HTTP non-200 error responses returned by TemboPlus
// Example: {"statusCode":401, "reason":"INVALID_CREDENTIALS"}
type APIError struct {
    StatusCode int    `json:"statusCode"`
    Reason     string `json:"reason,omitempty"`
    Message    string `json:"message,omitempty"`
    Details    json.RawMessage `json:"details,omitempty"`
}

func (e APIError) Error() string {
    if e.Reason != "" {
        return fmt.Sprintf("TemboPlus API Error [%d]: %s", e.StatusCode, e.Reason)
    }
    if e.Message != "" {
        return fmt.Sprintf("TemboPlus API Error [%d]: %s", e.StatusCode, e.Message)
    }
    return fmt.Sprintf("TemboPlus API Error [%d]", e.StatusCode)
}

// CollectionBalanceResponse represents the response for collection account balance
type CollectionBalanceResponse struct {
    AvailableBalance float64 `json:"availableBalance"`
    CurrentBalance   float64 `json:"currentBalance"`
    AccountNo        string  `json:"accountNo"`
    AccountStatus    string  `json:"accountStatus"`
    AccountName      string  `json:"accountName"`
}

// NullableFloat64 handles numbers that may be returned as number, string, or empty string
type NullableFloat64 struct {
    Value *float64
}

func (n *NullableFloat64) UnmarshalJSON(b []byte) error {
    // null or empty string
    if string(b) == "null" || string(b) == "\"\"" {
        n.Value = nil
        return nil
    }
    // Try number first
    var num float64
    if err := json.Unmarshal(b, &num); err == nil {
        n.Value = &num
        return nil
    }
    // Try string number
    var s string
    if err := json.Unmarshal(b, &s); err == nil {
        if s == "" {
            n.Value = nil
            return nil
        }
        var parsed float64
        if err2 := json.Unmarshal([]byte(s), &parsed); err2 == nil {
            n.Value = &parsed
            return nil
        }
        // If not a pure JSON number string, attempt ParseFloat
        if v, err3 := parseFloatFromString(s); err3 == nil {
            n.Value = &v
            return nil
        }
    }
    // As a fallback, ignore and set nil
    n.Value = nil
    return nil
}

// Helper used by NullableFloat64
func parseFloatFromString(s string) (float64, error) {
    // Use json to parse a quoted number safely
    var v float64
    return v, json.Unmarshal([]byte(s), &v)
}

// CollectionStatementRequest represents the request body for fetching a collection statement
type CollectionStatementRequest struct {
    StartDate string `json:"startDate"`
    EndDate   string `json:"endDate"`
    WalletID  string `json:"walletId,omitempty"`
}

// CollectionStatementEntry represents a single line item in the statement
type CollectionStatementEntry struct {
    AccountNo      string           `json:"accountNo"`
    DebitOrCredit  string           `json:"debitOrCredit"`
    TranRefNo      string           `json:"tranRefNo"`
    Narration      string           `json:"narration"`
    TxnDate        string           `json:"txnDate"`
    ValueDate      string           `json:"valueDate"`
    AmountCredited NullableFloat64  `json:"amountCredited"`
    AmountDebited  NullableFloat64  `json:"amountDebited"`
    Balance        float64          `json:"balance"`
}

// WalletToMobileRequest represents a wallet-to-mobile disbursement request
type WalletToMobileRequest struct {
    CountryCode     string  `json:"countryCode"`     // e.g., TZ
    AccountNo       string  `json:"accountNo"`       // Source wallet account number
    ServiceCode     string  `json:"serviceCode"`     // TZ-TIGO-B2C, TZ-AIRTEL-B2C
    Amount          float64 `json:"amount"`          // Amount to transfer
    MSISDN          string  `json:"msisdn"`          // Recipient MSISDN
    Narration       string  `json:"narration"`       // Transfer narration
    CurrencyCode    string  `json:"currencyCode"`    // e.g., TZS
    RecipientNames  string  `json:"recipientNames"`  // Recipient first and last names
    TransactionRef  string  `json:"transactionRef"`  // Your system reference
    TransactionDate string  `json:"transactionDate"` // Value date
    CallbackURL     string  `json:"callbackUrl"`     // Webhook URL
}

// PaymentStatusRequest represents the request body for checking payment status
type PaymentStatusRequest struct {
    TransactionRef string `json:"transactionRef,omitempty"`
    TransactionID  string `json:"transactionId,omitempty"`
}
