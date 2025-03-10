package models

const (
	LamportsPerSOL          = 1_000_000_000
	MaxRetries              = 5
	Slippage                = 0.3
	TOKEN_DECIMALS          = 6
	PriorityFeeLamport      = 50000
	MONITOR_DURATION        = 30
	MONITOR_DURATION_TRADES = 15
	History                 = 10
	TakeProfitTarget1       = 90 //18,000
	TakeProfitPerc1         = 50
	TakeProfitTarget2       = 130 // 26,000
	TakeProfitPerc2         = 25
	TakeProfitTarget3       = 250 // 50,000
	TakeProfitPerc3         = 10
	TakeProfitTarget4       = 400
	TakeProfitPerc4         = 10
)

const (
	PositionAmount = 0.004
)

const (
	EntryMarketCap  = 35.00
	MinEntryHistory = 2
	MinReserveRatio = 0.75
	TokenStability  = 0.000005
	MaxEntryHistory = 20
)

const (
	ExitMarketCap = 45
)

const (
	TOKEN_EXISTS    = "TOKEN ALREADY EXISTS IN ACCOUNT"
	NO_TOKEN_EXISTS = "TOKEN DOES NOT EXIST IN ACCOUNT"
)
