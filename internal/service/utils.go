package service

import "math/rand/v2"

// jitter возвращает число, рассыпавшееся относительно value на случайный процент в пределах
// [1-minPercent, 1+maxPercent].
// Например, если minPercent=0.15, maxPercent=0.15, получим диапазон [0.85*value, 1.15*value].
//
// minPercent и maxPercent должны быть >= 0 (0.1 = 10%). Если указано иное, значение выставится в 0.15.
func jitter(value, minPercent, maxPercent float64) float64 {
	if minPercent < 0 || maxPercent < 0 {
		minPercent = 0.15
		maxPercent = 0.15
	}
	factor := 1 - minPercent + rand.Float64()*(minPercent+maxPercent) // nolint:gosec
	return value * factor
}
