package service

import (
	"fmt"
	"sync"
)

var ErrInvalidSuccessRate = fmt.Errorf("success rate must be between 0.0 and 1.0")

// PaymentRateManager управляет текущим success rate для симуляции платежей
// Позволяет динамически менять rate через API (для UI)
type PaymentRateManager struct {
	mu          sync.RWMutex
	successRate float64
}

// NewPaymentRateManager создает менеджер с дефолтным rate
func NewPaymentRateManager(defaultRate float64) *PaymentRateManager {
	return &PaymentRateManager{
		successRate: defaultRate,
	}
}

// GetSuccessRate возвращает текущий success rate (потокобезопасно)
func (m *PaymentRateManager) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.successRate
}

// SetSuccessRate устанавливает новый success rate (потокобезопасно)
// rate должен быть в диапазоне [0.0, 1.0]
func (m *PaymentRateManager) SetSuccessRate(rate float64) error {
	if rate < 0.0 || rate > 1.0 {
		return ErrInvalidSuccessRate
	}

	m.mu.Lock()
	m.successRate = rate
	m.mu.Unlock()

	return nil
}

// ErrInvalidSuccessRate возвращается при попытке установить rate вне диапазона [0, 1]
