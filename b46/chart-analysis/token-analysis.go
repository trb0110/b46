package analysis

import (
	"b46/b46/models"
	"math"
	"time"
)

// ---------------------
// Basic Statistics on []float64
// ---------------------

// CalculateVolatility returns the standard deviation of the prices.
func CalculateVolatility(prices []float64) float64 {
	avg := CalculateMovingAverage(prices)
	var sumSquares float64
	for _, p := range prices {
		diff := p - avg
		sumSquares += diff * diff
	}
	if len(prices) == 0 {
		return 0
	}
	variance := sumSquares / float64(len(prices))
	return math.Sqrt(variance)
}

// CalculatePriceConvergence returns the difference between the maximum and minimum price.
func CalculatePriceConvergence(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	return MaxPrice(prices) - MinPrice(prices)
}

// CalculateMovingAverage returns the average value of the slice.
// Returns 0 if prices is empty.
func CalculateMovingAverage(prices []float64) float64 {
	n := len(prices)
	if n == 0 {
		return 0
	}
	var sum float64
	for _, p := range prices {
		sum += p
	}
	return sum / float64(n)
}

// CalculateVariance returns the variance of the values in prices given the provided average.
// Returns 0 if prices is empty.
func CalculateVariance(prices []float64, avg float64) float64 {
	n := len(prices)
	if n == 0 {
		return 0
	}
	var sum float64
	for _, p := range prices {
		diff := p - avg
		sum += diff * diff
	}
	return sum / float64(n)
}

// CalculateStandardDeviation returns the standard deviation (volatility) of the values in prices.
// Returns 0 if prices is empty.
func CalculateStandardDeviation(prices []float64) float64 {
	avg := CalculateMovingAverage(prices)
	return math.Sqrt(CalculateVariance(prices, avg))
}

// CalculatePriceTrendSlope computes the slope from a linear regression on the prices.
// Here the independent variable is the index (0,1,...,n-1).
// Returns 0 if there is insufficient data or if the denominator is zero.
func CalculatePriceTrendSlope(prices []float64) float64 {
	n := float64(len(prices))
	if n == 0 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i, p := range prices {
		x := float64(i)
		sumX += x
		sumY += p
		sumXY += x * p
		sumX2 += x * x
	}
	denom := n*sumX2 - math.Pow(sumX, 2)
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// IsConsistentlyTrendingUp returns true if each price is greater than or equal to the previous one.
func IsConsistentlyTrendingUp(prices []float64) bool {
	if len(prices) == 0 {
		return false
	}
	for i := 1; i < len(prices); i++ {
		if prices[i] < prices[i-1] {
			return false
		}
	}
	return true
}

// CalculatePercentageChange returns the percentage change from the first to the last value.
// If the slice is empty, returns 0.
func CalculatePercentageChange(prices []float64) float64 {
	n := len(prices)
	if n == 0 {
		return 0
	}
	first := prices[0]
	last := prices[n-1]
	// Protect against division by zero.
	if first == 0 {
		return 0
	}
	return ((last - first) / first) * 100.0
}

// MaxPrice returns the maximum value in prices.
// If prices is empty, returns 0.
func MaxPrice(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	maxVal := prices[0]
	for _, p := range prices {
		if p > maxVal {
			maxVal = p
		}
	}
	return maxVal
}

// MinPrice returns the minimum value in prices.
// If prices is empty, returns 0.
func MinPrice(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	minVal := prices[0]
	for _, p := range prices {
		if p < minVal {
			minVal = p
		}
	}
	return minVal
}

// ---------------------
// Time-Series Analysis (with timestamps)
// ---------------------

// CalculateTrendSlopeWithTimes computes the linear regression slope given prices and their timestamps.
// It converts the times into seconds elapsed since the first measurement.
// Returns 0 if prices is empty or if denominator is 0.
func CalculateTrendSlopeWithTimes(prices []float64, times []time.Time) float64 {
	n := len(prices)
	if n == 0 || len(times) != n {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	start := times[0].Unix()
	for i, p := range prices {
		// x is elapsed seconds since start
		x := float64(times[i].Unix() - start)
		sumX += x
		sumY += p
		sumXY += x * p
		sumX2 += x * x
	}
	denom := float64(n)*sumX2 - math.Pow(sumX, 2)
	if denom == 0 {
		return 0
	}
	return (float64(n)*sumXY - sumX*sumY) / denom
}

// ---------------------
// Reserve Ratios
// ---------------------

// CalculateReserveRatio returns the ratio of real to virtual token reserves.
// Returns 0 if virtual is 0.
func CalculateReserveRatio(real, virtual float64) float64 {
	if virtual == 0 {
		return 0
	}
	return real / virtual
}

// CalculateSolReserveRatio returns the ratio of real to virtual SOL reserves.
// Returns 0 if virtual is 0.
func CalculateSolReserveRatio(real, virtual float64) float64 {
	if virtual == 0 {
		return 0
	}
	return real / virtual
}

// ---------------------
// Trading/Valuation Functions
// ---------------------

// IsMarketCapSufficient checks if the token’s market cap meets or exceeds the threshold.
func IsMarketCapSufficient(info models.MemeInfo, minMarketCap float64) bool {
	return info.MarketCap >= minMarketCap
}

// HasSufficientReserves checks if the ratio of VirtualTokenReserves to RealTokenReserves
// in the token's bonding curve state exceeds minReserveRatio.
// (Note: Adjust this calculation if your intended ratio is different.)
func HasSufficientReserves(state *models.BondingCurveState, minReserveRatio float64) bool {
	if state.RealTokenReserves == 0 {
		return false
	}
	// Here we compute Virtual/Real; change to Real/Virtual if desired.
	ratio := float64(state.VirtualTokenReserves) / float64(state.RealTokenReserves)
	return ratio >= minReserveRatio
}

// IsPriceStable checks whether the variance of TokenPrice in the snapshots is below maxVariance.
func IsPriceStable(infos []models.MemeInfo, maxVariance float64) bool {
	n := len(infos)
	if n == 0 {
		return false
	}
	prices := make([]float64, n)
	for i, info := range infos {
		prices[i] = info.TokenPrice
	}
	avg := CalculateMovingAverage(prices)
	variance := CalculateVariance(prices, avg)
	return variance <= maxVariance
}

// ComputeTheoreticalPrice calculates a theoretical token price based on bonding curve data.
// For example, if your formula is:
//
//	theoretical = (virtualSolReserves / 10^(solDecimals)) / (virtualTokenReserves / 10^(tokenDecimals))
func ComputeTheoreticalPrice(state *models.BondingCurveState, virtualSolReserves uint64, solDecimals, tokenDecimals int) float64 {
	if state.VirtualTokenReserves == 0 {
		return 0.0
	}
	solValue := float64(virtualSolReserves) / float64(solDecimals)
	tokenValue := float64(state.VirtualTokenReserves) / math.Pow10(tokenDecimals)
	return solValue / tokenValue
}

// IsUndervalued checks if the token is undervalued relative to its theoretical price.
// discountThreshold is a fraction (e.g., 0.10 for 10% discount).
func IsUndervalued(currentPrice, theoreticalPrice, discountThreshold float64) bool {
	if theoreticalPrice == 0 {
		return false
	}
	discount := (theoreticalPrice - currentPrice) / theoreticalPrice
	return discount >= discountThreshold
}

// ComputePriceTrend aggregates the token prices from snapshots (assumed ordered chronologically)
// and returns the linear regression slope (using the index as the x-axis).
func ComputePriceTrend(infos []models.MemeInfo) float64 {
	n := len(infos)
	if n == 0 {
		return 0
	}
	prices := make([]float64, n)
	for i, info := range infos {
		prices[i] = info.TokenPrice
	}
	return CalculatePriceTrendSlope(prices)
}

// IsTrendingUp checks if the token price is trending upward at least by minSlope.
func IsTrendingUp(infos []models.MemeInfo, minSlope float64) bool {
	return ComputePriceTrend(infos) >= minSlope
}

// ComputeRiskRewardScore computes a risk-reward score using the current and theoretical prices
// and the volatility (standard deviation). Returns 0 if volatility or theoreticalPrice is 0.
func ComputeRiskRewardScore(currentPrice, theoreticalPrice, volatility float64) float64 {
	if volatility == 0 || theoreticalPrice == 0 {
		return 0
	}
	discount := (theoreticalPrice - currentPrice) / theoreticalPrice
	return discount / volatility
}

// AnalyzeTokenData computes various analysis metrics for a token using the given snapshots.
// It returns a TokenAnalysis struct with the computed values.
func AnalyzeTokenData(token models.MemeToken) models.TokenAnalysis {
	snapshots := token.Info
	n := len(snapshots)
	if n == 0 {
		return models.TokenAnalysis{}
	}

	// Extract prices from snapshots.
	prices := make([]float64, 0, n)
	for _, snap := range snapshots {
		prices = append(prices, snap.TokenPrice)
	}

	// Use the last snapshot for reserve data and market cap.
	lastSnap := snapshots[n-1]
	realTokenReserves := float64(lastSnap.BondingState.RealTokenReserves)
	virtualTokenReserves := float64(lastSnap.BondingState.VirtualTokenReserves)
	realSolReserves := float64(lastSnap.BondingState.RealSolReserves)
	virtualSolReserves := float64(lastSnap.BondingState.VirtualSolReserves)

	// Calculate basic metrics.
	avgPrice := CalculateMovingAverage(prices)
	volatility := CalculateStandardDeviation(prices)
	trendSlope := CalculatePriceTrendSlope(prices)
	percentageChange := CalculatePercentageChange(prices)
	reserveRatio := CalculateReserveRatio(realTokenReserves, virtualTokenReserves)
	solReserveRatio := CalculateSolReserveRatio(realSolReserves, virtualSolReserves)
	minPrice := MinPrice(prices)
	maxPrice := MaxPrice(prices)
	priceConvergence := maxPrice - minPrice
	consistentlyTrendingUp := IsConsistentlyTrendingUp(prices)

	// Compute a theoretical price.
	// (Assume solDecimals and tokenDecimals are known; adjust as needed.)
	solDecimals := models.LamportsPerSOL
	tokenDecimals := models.TOKEN_DECIMALS
	theoreticalPrice := ComputeTheoreticalPrice(lastSnap.BondingState, lastSnap.BondingState.VirtualSolReserves, solDecimals, tokenDecimals)

	// Compute risk-reward score.
	riskRewardScore := ComputeRiskRewardScore(prices[n-1], theoreticalPrice, volatility)

	// Check market cap and reserve sufficiency against thresholds (example thresholds).
	minMarketCap := models.EntryMarketCap     // example threshold for market cap
	minReserveRatio := models.MinReserveRatio // example threshold for reserve ratio
	marketCapSufficiency := lastSnap.MarketCap >= minMarketCap
	reservesSufficiency := reserveRatio >= minReserveRatio

	// Define a price stability threshold (example).
	priceStability := volatility < models.TokenStability

	t := time.Now()
	// Assemble the analysis result.
	analysis := models.TokenAnalysis{
		// Basic liquidity & valuation criteria:
		MarketCapSufficiency: marketCapSufficiency,
		ReservesSufficiency:  reservesSufficiency,
		DataPoints:           n,

		// Liquidity/reserve indicators:
		ReserveRatio:    reserveRatio,
		SolReserveRatio: solReserveRatio,

		// Price behavior over time:
		PriceStability:         priceStability,
		PriceConvergence:       priceConvergence,
		SimpleMovingAverage:    avgPrice,
		PercentageChange:       percentageChange,
		PriceTrendSlope:        trendSlope,
		ConsistentlyTrendingUp: consistentlyTrendingUp,

		// Risk / reward metrics:
		Volatility:       volatility,
		TheoreticalPrice: theoreticalPrice,
		CurrentPrice:     prices[n-1],
		RiskRewardScore:  riskRewardScore,
		Snapshot:         t,
		// Additional fields if desired:
		// TokenID, TokenName, Timestamp, MarketCap, etc.
		// (Make sure your models.TokenAnalysis struct includes these fields.)
		// For example:
		// TokenID:    tokenID,
		// TokenName:  tokenName,
		// Timestamp:  timestamp,
		// MarketCap:  lastSnap.MarketCap,
	}

	return analysis
}

// extractTimestamps is provided here for completeness.
// If your MemeInfo snapshots had a timestamp field (e.g. s.Time), you could extract them.
// Since the provided MemeInfo struct does not include a timestamp, this function
// is currently not used. You can remove it or modify it as needed.
func extractTimestamps(snapshots []models.MemeInfo) []time.Time {
	times := make([]time.Time, 0, len(snapshots))
	// If MemeInfo had a field like "Time time.Time", you would do:
	// for _, s := range snapshots {
	//     times = append(times, s.Time)
	// }
	// Otherwise, if the snapshots are evenly spaced, you might generate synthetic times.
	return times
}
