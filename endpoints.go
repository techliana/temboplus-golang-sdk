package temboplus

// API endpoints and defaults
const (
	// Default base URL for sandbox environment
	DefaultBaseURLSandbox = "https://sandbox.temboplus.com"
	// Default base URL for production environment
	DefaultBaseURLProduction = "https://api.temboplus.com"

	// Collection operations
	EndpointCollection       = "/tembo/v1/collection"
	EndpointCollectionStatus = "/tembo/v1/collection/status"

	// Wallet operations
	EndpointWalletCollectionBalance   = "/tembo/v1/wallet/collection-balance"
	EndpointWalletCollectionStatement = "/tembo/v1/wallet/collection-statement"
	EndpointWalletMainBalance         = "/tembo/v1/wallet/main-balance"
	EndpointWalletMainStatement       = "/tembo/v1/wallet/main-statement"

	// Payments
	EndpointPaymentWalletToMobile = "/tembo/v1/payment/wallet-to-mobile"
	EndpointPaymentStatus         = "/tembo/v1/payment/status"
)
