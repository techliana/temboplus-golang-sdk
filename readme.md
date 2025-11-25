# TemboPlus Go SDK

A comprehensive Go SDK for integrating with the TemboPlus payment platform, enabling mobile money collections, USSD payments, and financial services across Africa.

## Features

- **Money Collection**: Initiate single and bulk payment collections
- **Mobile Money Support**: Support for M-Pesa, Airtel Money, MTN Mobile Money, and Tigo Pesa
- **USSD Payments**: Generate USSD codes for feature phone users
- **Transaction Management**: Track, monitor, and manage payment statuses
- **Webhook Handling**: Process real-time payment notifications
- **Bulk Operations**: Handle multiple transactions efficiently
- **Error Handling**: Comprehensive error handling with detailed error messages
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/techliana/temboplus-golang-sdk
```

## Quick Start

### 1. Initialize the Client

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/yourusername/temboplus-go"
)

func main() {
    client := temboplus.NewClient(temboplus.ClientConfig{
        BaseURL:   "https://sandbox-api.temboplus.com", // Use sandbox for testing
        APIKey:    "your-api-key",
        APISecret: "your-api-secret",
        Timeout:   30 * time.Second,
    })
    
    // Your code here...
}
```

### 2. Collect Money

```go
ctx := context.Background()

request := temboplus.CollectMoneyRequest{
    Amount:        1000.0,
    Currency:      temboplus.CurrencyKES,
    PayerPhone:    temboplus.FormatPhoneNumber("0712345678", "254"),
    PayerName:     "John Doe",
    Description:   "Payment for goods",
    Reference:     temboplus.GenerateReference("ORDER"),
    CallbackURL:   "https://your-app.com/webhooks/temboplus",
    PaymentMethod: temboplus.PaymentMethodMobileMoney,
    Provider:      temboplus.ProviderMPESA,
}

response, err := client.CollectMoney(ctx, request)
if err != nil {
    log.Fatal(err)
}

log.Printf("Transaction ID: %s", response.TransactionID)
log.Printf("Status: % s", response.Status)
```
