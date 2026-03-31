package signal

import (
	"fmt"
	"math"
)

// strengthScaleFactorSMA scales the SMA distance to a 0-1 strength value.
// A 5% relative difference between fast and slow SMA maps to strength 1.0.
const strengthScaleFactorSMA = 20.0
// Returns: value (BULLISH/BEARISH/NEUTRAL), strength (0.0-1.0), detail string.
func ComputeSMACross(prices []float64, fastPeriod, slowPeriod int) (string, float64, string) {
	if len(prices) < slowPeriod+1 {
		return "NEUTRAL", 0.0, fmt.Sprintf("insufficient data (need %d, have %d)", slowPeriod+1, len(prices))
	}

	fastNow := sma(prices, fastPeriod)
	slowNow := sma(prices, slowPeriod)

	// Previous bar values
	prevPrices := prices[:len(prices)-1]
	fastPrev := sma(prevPrices, fastPeriod)
	slowPrev := sma(prevPrices, slowPeriod)

	// Detect crossover
	crossedAbove := fastPrev <= slowPrev && fastNow > slowNow
	crossedBelow := fastPrev >= slowPrev && fastNow < slowNow

	// Strength: relative distance between SMAs
	diff := math.Abs(fastNow-slowNow) / slowNow
	strength := math.Min(1.0, diff*strengthScaleFactorSMA)

	if crossedAbove {
		return "BULLISH", strength, fmt.Sprintf("SMA%d (%.2f) crossed above SMA%d (%.2f)", fastPeriod, fastNow, slowPeriod, slowNow)
	}
	if crossedBelow {
		return "BEARISH", strength, fmt.Sprintf("SMA%d (%.2f) crossed below SMA%d (%.2f)", fastPeriod, fastNow, slowPeriod, slowNow)
	}

	if fastNow > slowNow {
		return "BULLISH", strength * 0.5, fmt.Sprintf("SMA%d (%.2f) above SMA%d (%.2f), no fresh cross", fastPeriod, fastNow, slowPeriod, slowNow)
	}
	return "BEARISH", strength * 0.5, fmt.Sprintf("SMA%d (%.2f) below SMA%d (%.2f), no fresh cross", fastPeriod, fastNow, slowPeriod, slowNow)
}

// ComputeRSI computes the Relative Strength Index signal.
// Returns: value (BULLISH/BEARISH/NEUTRAL), strength (0.0-1.0), detail string.
func ComputeRSI(prices []float64, period int, overbought, oversold float64) (string, float64, string) {
	if len(prices) < period+1 {
		return "NEUTRAL", 0.0, fmt.Sprintf("insufficient data (need %d, have %d)", period+1, len(prices))
	}

	rsiVal := rsi(prices, period)
	detail := fmt.Sprintf("RSI(%d) = %.1f", period, rsiVal)

	if rsiVal >= overbought {
		strength := math.Min(1.0, (rsiVal-overbought)/(100-overbought))
		return "BEARISH", strength, detail + fmt.Sprintf(" (overbought > %.0f)", overbought)
	}
	if rsiVal <= oversold {
		strength := math.Min(1.0, (oversold-rsiVal)/oversold)
		return "BULLISH", strength, detail + fmt.Sprintf(" (oversold < %.0f)", oversold)
	}

	// Neutral: strength represents distance from midpoint
	mid := (overbought + oversold) / 2
	strength := math.Abs(rsiVal-mid) / (overbought - mid)
	return "NEUTRAL", strength, detail
}

// ComputePriceChange computes the price change signal over a window.
// Returns: value (BULLISH/BEARISH/NEUTRAL), strength (0.0-1.0), detail string.
func ComputePriceChange(prices []float64, thresholdPct float64, window int) (string, float64, string) {
	if len(prices) < window+1 {
		return "NEUTRAL", 0.0, fmt.Sprintf("insufficient data (need %d, have %d)", window+1, len(prices))
	}

	current := prices[len(prices)-1]
	past := prices[len(prices)-1-window]

	if past == 0 {
		return "NEUTRAL", 0.0, "base price is zero"
	}

	changePct := (current - past) / past * 100
	detail := fmt.Sprintf("price change %.2f%% over %d bars (%.2f → %.2f)", changePct, window, past, current)

	absChange := math.Abs(changePct)
	strength := math.Min(1.0, absChange/thresholdPct)

	if changePct >= thresholdPct {
		return "BULLISH", strength, detail
	}
	if changePct <= -thresholdPct {
		return "BEARISH", strength, detail
	}
	return "NEUTRAL", strength, detail
}

// sma computes the Simple Moving Average of the last n prices.
func sma(prices []float64, n int) float64 {
	if len(prices) < n {
		return 0
	}
	sum := 0.0
	for _, p := range prices[len(prices)-n:] {
		sum += p
	}
	return sum / float64(n)
}

// rsi computes the RSI of the last period+1 prices using the Wilder method.
func rsi(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50
	}

	gains := 0.0
	losses := 0.0

	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
