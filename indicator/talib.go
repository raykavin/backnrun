package indicator

import "github.com/markcheno/go-talib"

// MaType represents moving average type
type MaType = talib.MaType

// Moving average type constants
const (
	TypeSMA   = talib.SMA   // Simple Moving Average
	TypeEMA   = talib.EMA   // Exponential Moving Average
	TypeWMA   = talib.WMA   // Weighted Moving Average
	TypeDEMA  = talib.DEMA  // Double Exponential Moving Average
	TypeTEMA  = talib.TEMA  // Triple Exponential Moving Average
	TypeTRIMA = talib.TRIMA // Triangular Moving Average
	TypeKAMA  = talib.KAMA  // Kaufman Adaptive Moving Average
	TypeMAMA  = talib.MAMA  // MESA Adaptive Moving Average
	TypeT3MA  = talib.T3MA  // Triple Exponential Moving Average with T3 smoothing
)

// ------------------------------------------
// Overlap Studies (Moving Averages, Bands)
// ------------------------------------------

// BB calculates Bollinger Bands
// Returns upper, middle, and lower bands
func BB(input []float64, period int, deviation float64, maType MaType) ([]float64, []float64, []float64) {
	return talib.BBands(input, period, deviation, deviation, maType)
}

// DEMA calculates Double Exponential Moving Average
func DEMA(input []float64, period int) []float64 {
	return talib.Dema(input, period)
}

// EMA calculates Exponential Moving Average
func EMA(input []float64, period int) []float64 {
	return talib.Ema(input, period)
}

// HTTrendline calculates Hilbert Transform - Instantaneous Trendline
func HTTrendline(input []float64) []float64 {
	return talib.HtTrendline(input)
}

// KAMA calculates Kaufman Adaptive Moving Average
func KAMA(input []float64, period int) []float64 {
	return talib.Kama(input, period)
}

// MA calculates Moving Average with specified type
func MA(input []float64, period int, maType MaType) []float64 {
	return talib.Ma(input, period, maType)
}

// MAMA calculates MESA Adaptive Moving Average
// Returns MAMA and FAMA (Following Adaptive Moving Average)
func MAMA(input []float64, fastLimit float64, slowLimit float64) ([]float64, []float64) {
	return talib.Mama(input, fastLimit, slowLimit)
}

// MaVp calculates Moving Average with Variable Period
func MaVp(input []float64, periods []float64, minPeriod int, maxPeriod int, maType MaType) []float64 {
	return talib.MaVp(input, periods, minPeriod, maxPeriod, maType)
}

// MidPoint calculates MidPoint over period
func MidPoint(input []float64, period int) []float64 {
	return talib.MidPoint(input, period)
}

// MidPrice calculates MidPrice over period
func MidPrice(high []float64, low []float64, period int) []float64 {
	return talib.MidPrice(high, low, period)
}

// SAR calculates Parabolic SAR (Stop And Reverse)
func SAR(high []float64, low []float64, acceleration float64, maximum float64) []float64 {
	return talib.Sar(high, low, acceleration, maximum)
}

// SARExt calculates Parabolic SAR - Extended
func SARExt(high []float64, low []float64,
	startValue float64,
	offsetOnReverse float64,
	accelerationInitLong float64,
	accelerationLong float64,
	accelerationMaxLong float64,
	accelerationInitShort float64,
	accelerationShort float64,
	accelerationMaxShort float64) []float64 {
	return talib.SarExt(high, low, startValue, offsetOnReverse, accelerationInitLong, accelerationLong,
		accelerationMaxLong, accelerationInitShort, accelerationShort, accelerationMaxShort)
}

// SMA calculates Simple Moving Average
func SMA(input []float64, period int) []float64 {
	return talib.Sma(input, period)
}

// T3 calculates Triple Exponential Moving Average (T3)
func T3(input []float64, period int, vFactor float64) []float64 {
	return talib.T3(input, period, vFactor)
}

// TEMA calculates Triple Exponential Moving Average
func TEMA(input []float64, period int) []float64 {
	return talib.Tema(input, period)
}

// TRIMA calculates Triangular Moving Average
func TRIMA(input []float64, period int) []float64 {
	return talib.Trima(input, period)
}

// WMA calculates Weighted Moving Average
func WMA(input []float64, period int) []float64 {
	return talib.Wma(input, period)
}

// ---------------------------------------
// Momentum Indicators
// ---------------------------------------

// ADX calculates Average Directional Movement Index
func ADX(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Adx(high, low, close, period)
}

// ADXR calculates Average Directional Movement Index Rating
func ADXR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.AdxR(high, low, close, period)
}

// APO calculates Absolute Price Oscillator
func APO(input []float64, fastPeriod int, slowPeriod int, maType MaType) []float64 {
	return talib.Apo(input, fastPeriod, slowPeriod, maType)
}

// Aroon calculates Aroon indicator
// Returns aroonDown and aroonUp
func Aroon(high []float64, low []float64, period int) ([]float64, []float64) {
	return talib.Aroon(high, low, period)
}

// AroonOsc calculates Aroon Oscillator
func AroonOsc(high []float64, low []float64, period int) []float64 {
	return talib.AroonOsc(high, low, period)
}

// BOP calculates Balance Of Power
func BOP(open []float64, high []float64, low []float64, close []float64) []float64 {
	return talib.Bop(open, high, low, close)
}

// CMO calculates Chande Momentum Oscillator
func CMO(input []float64, period int) []float64 {
	return talib.Cmo(input, period)
}

// CCI calculates Commodity Channel Index
func CCI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Cci(high, low, close, period)
}

// DX calculates Directional Movement Index
func DX(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Dx(high, low, close, period)
}

// MACD calculates Moving Average Convergence/Divergence
// Returns MACD, signal, and histogram
func MACD(input []float64, fastPeriod int, slowPeriod int, signalPeriod int) ([]float64, []float64, []float64) {
	return talib.Macd(input, fastPeriod, slowPeriod, signalPeriod)
}

// MACDExt calculates MACD with controllable MA types
// Returns MACD, signal, and histogram
func MACDExt(input []float64, fastPeriod int, fastMAType MaType, slowPeriod int, slowMAType MaType,
	signalPeriod int, signalMAType MaType) ([]float64, []float64, []float64) {
	return talib.MacdExt(input, fastPeriod, fastMAType, slowPeriod, slowMAType, signalPeriod, signalMAType)
}

// MACDFix calculates MACD with fixed 12/26 periods
// Returns MACD, signal, and histogram
func MACDFix(input []float64, signalPeriod int) ([]float64, []float64, []float64) {
	return talib.MacdFix(input, signalPeriod)
}

// MinusDI calculates Minus Directional Indicator
func MinusDI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.MinusDI(high, low, close, period)
}

// MinusDM calculates Minus Directional Movement
func MinusDM(high []float64, low []float64, period int) []float64 {
	return talib.MinusDM(high, low, period)
}

// MFI calculates Money Flow Index
func MFI(high []float64, low []float64, close []float64, volume []float64, period int) []float64 {
	return talib.Mfi(high, low, close, volume, period)
}

// Momentum calculates momentum indicator
func Momentum(input []float64, period int) []float64 {
	return talib.Mom(input, period)
}

// PlusDI calculates Plus Directional Indicator
func PlusDI(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.PlusDI(high, low, close, period)
}

// PlusDM calculates Plus Directional Movement
func PlusDM(high []float64, low []float64, period int) []float64 {
	return talib.PlusDM(high, low, period)
}

// PPO calculates Percentage Price Oscillator
func PPO(input []float64, fastPeriod int, slowPeriod int, maType MaType) []float64 {
	return talib.Ppo(input, fastPeriod, slowPeriod, maType)
}

// ROCP calculates Rate of Change Percentage: (price-prevPrice)/prevPrice
func ROCP(input []float64, period int) []float64 {
	return talib.Rocp(input, period)
}

// ROC calculates Rate of Change: ((price/prevPrice)-1)*100
func ROC(input []float64, period int) []float64 {
	return talib.Roc(input, period)
}

// ROCR calculates Rate of Change Ratio: (price/prevPrice)
func ROCR(input []float64, period int) []float64 {
	return talib.Rocr(input, period)
}

// ROCR100 calculates Rate of Change Ratio 100 scale: (price/prevPrice)*100
func ROCR100(input []float64, period int) []float64 {
	return talib.Rocr100(input, period)
}

// RSI calculates Relative Strength Index
func RSI(input []float64, period int) []float64 {
	return talib.Rsi(input, period)
}

// Stoch calculates Slow Stochastic Indicator
// Returns slowK and slowD
func Stoch(high []float64, low []float64, close []float64, fastKPeriod int, slowKPeriod int,
	slowKMAType MaType, slowDPeriod int, slowDMAType MaType) ([]float64, []float64) {
	return talib.Stoch(high, low, close, fastKPeriod, slowKPeriod, slowKMAType, slowDPeriod, slowDMAType)
}

// StochF calculates Fast Stochastic Indicator
// Returns fastK and fastD
func StochF(high []float64, low []float64, close []float64, fastKPeriod int, fastDPeriod int,
	fastDMAType MaType) ([]float64, []float64) {
	return talib.StochF(high, low, close, fastKPeriod, fastDPeriod, fastDMAType)
}

// StochRSI calculates Stochastic RSI
// Returns fastK and fastD
func StochRSI(input []float64, period int, fastKPeriod int, fastDPeriod int, fastDMAType MaType) ([]float64, []float64) {
	return talib.StochRsi(input, period, fastKPeriod, fastDPeriod, fastDMAType)
}

// Trix calculates TRIX - 1-day Rate-of-Change of a Triple Smooth EMA
func Trix(input []float64, period int) []float64 {
	return talib.Trix(input, period)
}

// UltOsc calculates Ultimate Oscillator
func UltOsc(high []float64, low []float64, close []float64, period1 int, period2 int, period3 int) []float64 {
	return talib.UltOsc(high, low, close, period1, period2, period3)
}

// WilliamsR calculates Williams' %R
func WilliamsR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.WillR(high, low, close, period)
}

// ---------------------------------------
// Volume Indicators
// ---------------------------------------

// Ad calculates Chaikin A/D Line
func Ad(high []float64, low []float64, close []float64, volume []float64) []float64 {
	return talib.Ad(high, low, close, volume)
}

// AdOsc calculates Chaikin A/D Oscillator
func AdOsc(high []float64, low []float64, close []float64, volume []float64, fastPeriod int, slowPeriod int) []float64 {
	return talib.AdOsc(high, low, close, volume, fastPeriod, slowPeriod)
}

// OBV calculates On Balance Volume
func OBV(input []float64, volume []float64) []float64 {
	return talib.Obv(input, volume)
}

// ---------------------------------------
// Volatility Indicators
// ---------------------------------------

// ATR calculates Average True Range
func ATR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Atr(high, low, close, period)
}

// NATR calculates Normalized Average True Range
func NATR(high []float64, low []float64, close []float64, period int) []float64 {
	return talib.Natr(high, low, close, period)
}

// TRANGE calculates True Range
func TRANGE(high []float64, low []float64, close []float64) []float64 {
	return talib.TRange(high, low, close)
}

// ---------------------------------------
// Price Transform Functions
// ---------------------------------------

// AvgPrice calculates Average Price
func AvgPrice(open []float64, high []float64, low []float64, close []float64) []float64 {
	return talib.AvgPrice(open, high, low, close)
}

// MedPrice calculates Median Price
func MedPrice(high []float64, low []float64) []float64 {
	return talib.MedPrice(high, low)
}

// TypPrice calculates Typical Price
func TypPrice(high []float64, low []float64, close []float64) []float64 {
	return talib.TypPrice(high, low, close)
}

// WCLPrice calculates Weighted Close Price
func WCLPrice(high []float64, low []float64, close []float64) []float64 {
	return talib.WclPrice(high, low, close)
}

// ---------------------------------------
// Cycle Indicator Functions
// ---------------------------------------

// HTDcPeriod calculates Hilbert Transform - Dominant Cycle Period
func HTDcPeriod(input []float64) []float64 {
	return talib.HtDcPeriod(input)
}

// HTDcPhase calculates Hilbert Transform - Dominant Cycle Phase
func HTDcPhase(input []float64) []float64 {
	return talib.HtDcPhase(input)
}

// HTPhasor calculates Hilbert Transform - Phasor Components
// Returns inPhase and quadrature
func HTPhasor(input []float64) ([]float64, []float64) {
	return talib.HtPhasor(input)
}

// HTSine calculates Hilbert Transform - SineWave
// Returns sine and leadSine
func HTSine(input []float64) ([]float64, []float64) {
	return talib.HtSine(input)
}

// HTTrendMode calculates Hilbert Transform - Trend vs Cycle Mode
func HTTrendMode(input []float64) []float64 {
	return talib.HtTrendMode(input)
}

// ---------------------------------------
// Statistic Functions
// ---------------------------------------

// Beta calculates Beta
func Beta(input0 []float64, input1 []float64, period int) []float64 {
	return talib.Beta(input0, input1, period)
}

// Correl calculates Pearson's Correlation Coefficient (r)
func Correl(input0 []float64, input1 []float64, period int) []float64 {
	return talib.Correl(input0, input1, period)
}

// LinearReg calculates Linear Regression
func LinearReg(input []float64, period int) []float64 {
	return talib.LinearReg(input, period)
}

// LinearRegAngle calculates Linear Regression Angle
func LinearRegAngle(input []float64, period int) []float64 {
	return talib.LinearRegAngle(input, period)
}

// LinearRegIntercept calculates Linear Regression Intercept
func LinearRegIntercept(input []float64, period int) []float64 {
	return talib.LinearRegIntercept(input, period)
}

// LinearRegSlope calculates Linear Regression Slope
func LinearRegSlope(input []float64, period int) []float64 {
	return talib.LinearRegSlope(input, period)
}

// StdDev calculates Standard Deviation
func StdDev(input []float64, period int, nbDev float64) []float64 {
	return talib.StdDev(input, period, nbDev)
}

// TSF calculates Time Series Forecast
func TSF(input []float64, period int) []float64 {
	return talib.Tsf(input, period)
}

// Var calculates Variance
func Var(input []float64, period int) []float64 {
	return talib.Var(input, period)
}

// ---------------------------------------
// Math Transform Functions
// ---------------------------------------

// Acos calculates Arc Cosine
func Acos(input []float64) []float64 {
	return talib.Acos(input)
}

// Asin calculates Arc Sine
func Asin(input []float64) []float64 {
	return talib.Asin(input)
}

// Atan calculates Arc Tangent
func Atan(input []float64) []float64 {
	return talib.Atan(input)
}

// Ceil calculates Ceiling
func Ceil(input []float64) []float64 {
	return talib.Ceil(input)
}

// Cos calculates Cosine
func Cos(input []float64) []float64 {
	return talib.Cos(input)
}

// Cosh calculates Hyperbolic Cosine
func Cosh(input []float64) []float64 {
	return talib.Cosh(input)
}

// Exp calculates Exponential
func Exp(input []float64) []float64 {
	return talib.Exp(input)
}

// Floor calculates Floor
func Floor(input []float64) []float64 {
	return talib.Floor(input)
}

// Ln calculates Natural Log
func Ln(input []float64) []float64 {
	return talib.Ln(input)
}

// Log10 calculates Base-10 Log
func Log10(input []float64) []float64 {
	return talib.Log10(input)
}

// Sin calculates Sine
func Sin(input []float64) []float64 {
	return talib.Sin(input)
}

// Sinh calculates Hyperbolic Sine
func Sinh(input []float64) []float64 {
	return talib.Sinh(input)
}

// Sqrt calculates Square Root
func Sqrt(input []float64) []float64 {
	return talib.Sqrt(input)
}

// Tan calculates Tangent
func Tan(input []float64) []float64 {
	return talib.Tan(input)
}

// Tanh calculates Hyperbolic Tangent
func Tanh(input []float64) []float64 {
	return talib.Tanh(input)
}

// ---------------------------------------
// Math Operator Functions
// ---------------------------------------

// Add calculates Vector Arithmetic Add
func Add(input0, input1 []float64) []float64 {
	return talib.Add(input0, input1)
}

// Div calculates Vector Arithmetic Division
func Div(input0, input1 []float64) []float64 {
	return talib.Div(input0, input1)
}

// Max calculates Highest value over period
func Max(input []float64, period int) []float64 {
	return talib.Max(input, period)
}

// MaxIndex calculates Index of highest value over period
func MaxIndex(input []float64, period int) []float64 {
	return talib.MaxIndex(input, period)
}

// Min calculates Lowest value over period
func Min(input []float64, period int) []float64 {
	return talib.Min(input, period)
}

// MinIndex calculates Index of lowest value over period
func MinIndex(input []float64, period int) []float64 {
	return talib.MinIndex(input, period)
}

// MinMax calculates Lowest and highest values over period
// Returns min and max
func MinMax(input []float64, period int) ([]float64, []float64) {
	return talib.MinMax(input, period)
}

// MinMaxIndex calculates Indexes of lowest and highest values over period
// Returns minIndex and maxIndex
func MinMaxIndex(input []float64, period int) ([]float64, []float64) {
	return talib.MinMaxIndex(input, period)
}

// Mult calculates Vector Arithmetic Multiplication
func Mult(input0, input1 []float64) []float64 {
	return talib.Mult(input0, input1)
}

// Sub calculates Vector Arithmetic Subtraction
func Sub(input0, input1 []float64) []float64 {
	return talib.Sub(input0, input1)
}

// Sum calculates Summation over period
func Sum(input []float64, period int) []float64 {
	return talib.Sum(input, period)
}
