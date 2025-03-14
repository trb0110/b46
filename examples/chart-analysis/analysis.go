package trading

import (
	"math"
	"time"
)

type TokenAnalysis struct {
	// Basic token info:
	TokenID   string    // Unique token identifier
	TokenName string    // Human-readable name
	Timestamp time.Time // Time of the latest snapshot

	// Price and market metrics:
	CurrentPrice       float64 // Latest observed token price
	MarketCap          float64 // Latest market cap
	AveragePrice       float64 // Simple moving average over the period
	PriceChangePercent float64 // Percent change from first to last snapshot
	PriceTrendSlope    float64 // Slope from linear regression on price vs. time
	PriceVolatility    float64 // Standard deviation of token price

	// Liquidity/reserve indicators:
	ReserveRatio    float64 // RealTokenReserves / VirtualTokenReserves
	SolReserveRatio float64 // RealSolReserves / VirtualSolReserves (if applicable)

	// Qualitative flags:
	IsLiquid      bool    // e.g. true if reserve ratios exceed minimum thresholds
	IsPriceStable bool    // true if volatility is below a threshold
	AnomalyScore  float64 // A normalized score indicating unusual behavior
	DataPoints    int     // Number of snapshots analyzed
}

// BondingCurveState represents the state of a bonding curve
type BondingCurveState struct {
	VirtualTokenReserves uint64
	VirtualSolReserves   uint64
	RealTokenReserves    uint64
	RealSolReserves      uint64
	CurrentSupply        uint64
	Complete             bool
}

// Each snapshot of token data is represented as follows:
type MemeInfo struct {
	BondingState *BondingCurveState
	TokenPrice   float64
	MarketCap    float64
	Snapshot     time.Time
}

// --- Preexisting functions ---

// IsMarketCapSufficient checks that the token’s market cap is at least a given minimum.
func IsMarketCapSufficient(info MemeInfo, minMarketCap float64) bool {
	return info.MarketCap >= minMarketCap
}

// HasSufficientReserves checks that the ratio of VirtualTokenReserves to CurrentSupply
// meets or exceeds a minimum threshold.
func HasSufficientReserves(state *BondingCurveState, minReserveRatio float64) bool {
	if state.CurrentSupply == 0 {
		return false
	}
	ratio := float64(state.VirtualTokenReserves) / float64(state.CurrentSupply)
	return ratio >= minReserveRatio
}

// IsPriceStable computes the variance of TokenPrice values over a series of snapshots.
// It returns true if the variance is below a given maximum threshold.
func IsPriceStable(infos []MemeInfo, maxVariance float64) bool {
	n := len(infos)
	if n == 0 {
		return false
	}
	sum := 0.0
	for _, info := range infos {
		sum += info.TokenPrice
	}
	mean := sum / float64(n)
	var variance float64
	for _, info := range infos {
		diff := info.TokenPrice - mean
		variance += diff * diff
	}
	variance /= float64(n)
	return variance <= maxVariance
}

// ComputeTheoreticalPrice computes a theoretical price from the bonding curve.
// Here, we use a sample model: price = (virtualSolReserves / 10^(solDecimals)) / (virtualTokenReserves / 10^(tokenDecimals))
// (This is only an example; adjust the formula as needed.)
func ComputeTheoreticalPrice(state *BondingCurveState, virtualSolReserves uint64, solDecimals, tokenDecimals int) float64 {
	if state.VirtualTokenReserves == 0 {
		return 0.0
	}
	solValue := float64(virtualSolReserves) / math.Pow10(solDecimals)
	tokenValue := float64(state.VirtualTokenReserves) / math.Pow10(tokenDecimals)
	return solValue / tokenValue
}

// IsUndervalued checks if the current price is discounted relative to the theoretical price
// by at least discountThreshold (expressed as a fraction, e.g. 0.10 for 10%).
func IsUndervalued(currentPrice, theoreticalPrice, discountThreshold float64) bool {
	if theoreticalPrice == 0 {
		return false
	}
	discount := (theoreticalPrice - currentPrice) / theoreticalPrice
	return discount >= discountThreshold
}

// ComputePriceTrend performs a simple linear regression on TokenPrice values (assumed equally spaced in time)
// and returns the slope. A positive slope indicates an upward trend.
func ComputePriceTrend(infos []MemeInfo) float64 {
	n := len(infos)
	if n == 0 {
		return 0
	}
	var sumX, sumY, sumXY, sumXX float64
	for i, info := range infos {
		x := float64(i)
		y := info.TokenPrice
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	denom := float64(n)*sumXX - sumX*sumX
	if denom == 0 {
		return 0
	}
	slope := (float64(n)*sumXY - sumX*sumY) / denom
	return slope
}

// IsTrendingUp returns true if the computed price trend slope is at least minSlope.
func IsTrendingUp(infos []MemeInfo, minSlope float64) bool {
	slope := ComputePriceTrend(infos)
	return slope >= minSlope
}

// ComputeVolatility returns the standard deviation of TokenPrice values over a series of snapshots.
func ComputeVolatility(infos []MemeInfo) float64 {
	n := len(infos)
	if n == 0 {
		return 0
	}
	var sum float64
	for _, info := range infos {
		sum += info.TokenPrice
	}
	mean := sum / float64(n)
	var variance float64
	for _, info := range infos {
		diff := info.TokenPrice - mean
		variance += diff * diff
	}
	variance /= float64(n)
	return math.Sqrt(variance)
}

// ComputeRiskRewardScore computes a normalized score as the discount (relative to theoretical price)
// divided by the volatility. A higher score might indicate a more attractive opportunity.
func ComputeRiskRewardScore(currentPrice, theoreticalPrice, volatility float64) float64 {
	if volatility == 0 || theoreticalPrice == 0 {
		return 0
	}
	discount := (theoreticalPrice - currentPrice) / theoreticalPrice
	return discount / volatility
}

// --- Additional functions based on a series of snapshots ---

// ComputeSimpleMovingAverage computes the arithmetic average of TokenPrice over the snapshots.
func ComputeSimpleMovingAverage(infos []MemeInfo) float64 {
	n := len(infos)
	if n == 0 {
		return 0
	}
	sum := 0.0
	for _, info := range infos {
		sum += info.TokenPrice
	}
	return sum / float64(n)
}

// MinMaxPrice computes the minimum and maximum TokenPrice observed over the snapshots.
func MinMaxPrice(infos []MemeInfo) (min, max float64) {
	n := len(infos)
	if n == 0 {
		return 0, 0
	}
	min = infos[0].TokenPrice
	max = infos[0].TokenPrice
	for _, info := range infos {
		price := info.TokenPrice
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
	}
	return min, max
}

// IsPriceConverging returns true if the difference between the maximum and minimum TokenPrice
// in the snapshots is less than or equal to maxDelta. This can indicate that the price is converging.
func IsPriceConverging(infos []MemeInfo, maxDelta float64) bool {
	min, max := MinMaxPrice(infos)
	return (max - min) <= maxDelta
}

// ComputePercentageChange calculates the percentage change in TokenPrice between the first and last snapshot.
func ComputePercentageChange(infos []MemeInfo) float64 {
	n := len(infos)
	if n < 2 {
		return 0
	}
	first := infos[0].TokenPrice
	last := infos[n-1].TokenPrice
	if first == 0 {
		return 0
	}
	return (last - first) / first
}

// IsConsistentlyTrendingUp returns true if each successive snapshot shows an increase in TokenPrice.
func IsConsistentlyTrendingUp(infos []MemeInfo) bool {
	n := len(infos)
	if n < 2 {
		return false
	}
	for i := 1; i < n; i++ {
		if infos[i].TokenPrice < infos[i-1].TokenPrice {
			return false
		}
	}
	return true
}

// CalculateMovingAverage returns the average price over the period.
func CalculateMovingAverage(prices []float64) float64 {
	var sum float64
	for _, p := range prices {
		sum += p
	}
	return sum / float64(len(prices))
}

// CalculateVolatility returns the standard deviation of the prices.
func CalculateVolatility(prices []float64) float64 {
	avg := CalculateMovingAverage(prices)
	var sumSquares float64
	for _, p := range prices {
		diff := p - avg
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(prices))
	return math.Sqrt(variance)
}

// CalculateTrendSlope computes the linear regression slope given prices and their timestamps.
func CalculateTrendSlope(prices []float64, times []time.Time) float64 {
	// Convert times to seconds since the first measurement.
	n := float64(len(prices))
	if n == 0 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	start := times[0].Unix()
	for i, p := range prices {
		x := float64(times[i].Unix() - start)
		sumX += x
		sumY += p
		sumXY += x * p
		sumX2 += x * x
	}
	// Slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX^2)
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// CalculateReserveRatio returns the ratio of real to virtual token reserves.
func CalculateReserveRatio(real, virtual float64) float64 {
	if virtual == 0 {
		return 0
	}
	return real / virtual
}

// Similarly, you can add a function for the Sol reserve ratio.
func CalculateSolReserveRatio(real, virtual float64) float64 {
	if virtual == 0 {
		return 0
	}
	return real / virtual
}

func AnalyzeTokenData(tokenID, tokenName string, snapshots []MemeInfo, timestamp time.Time) TokenAnalysis {
	// Extract prices and reserve data from snapshots:
	var prices []float64
	// For simplicity, we use the values from the first snapshot for reserves.
	// In practice, you might average them or pick the latest.
	var realTokenReserves, virtualTokenReserves float64
	var realSolReserves, virtualSolReserves float64
	for i, snap := range snapshots {
		prices = append(prices, snap.TokenPrice)
		if i == len(snapshots)-1 {
			realTokenReserves = float64(snap.BondingState.RealTokenReserves)
			virtualTokenReserves = float64(snap.BondingState.VirtualTokenReserves)
			realSolReserves = float64(snap.BondingState.RealSolReserves)
			virtualSolReserves = float64(snap.BondingState.VirtualSolReserves)
		}
	}

	analysis := TokenAnalysis{
		TokenID:         tokenID,
		TokenName:       tokenName,
		Timestamp:       timestamp,
		CurrentPrice:    prices[len(prices)-1],
		MarketCap:       snapshots[len(snapshots)-1].MarketCap,
		AveragePrice:    CalculateMovingAverage(prices),
		PriceVolatility: CalculateVolatility(prices),
		PriceTrendSlope: CalculateTrendSlope(prices, extractTimestamps(snapshots)),
		DataPoints:      len(prices),
		ReserveRatio:    CalculateReserveRatio(realTokenReserves, virtualTokenReserves),
		SolReserveRatio: CalculateSolReserveRatio(realSolReserves, virtualSolReserves),
		// You can define criteria to set IsLiquid or IsPriceStable:
		IsLiquid:      CalculateReserveRatio(realTokenReserves, virtualTokenReserves) > 0.75, // for example
		IsPriceStable: CalculateVolatility(prices) < 0.000005,                                // example threshold
		// AnomalyScore could combine deviations in these metrics.
	}

	// Price change percent based on first and last snapshots:
	firstPrice := prices[0]
	lastPrice := prices[len(prices)-1]
	if firstPrice != 0 {
		analysis.PriceChangePercent = ((lastPrice - firstPrice) / firstPrice) * 100
	}

	return analysis
}

// Helper to extract timestamps from snapshots.
func extractTimestamps(snapshots []MemeInfo) []time.Time {
	var times []time.Time
	for _, s := range snapshots {
		times = append(times, s.Snapshot)
	}
	return times
}
