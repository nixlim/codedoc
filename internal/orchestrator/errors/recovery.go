package errors

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// RecoveryStrategy defines how to recover from an error.
type RecoveryStrategy interface {
	// CanRecover determines if recovery is possible
	CanRecover(err error) bool

	// Recover attempts to recover from the error
	Recover(ctx context.Context, err error) error

	// GetBackoffDuration returns the wait time before retry
	GetBackoffDuration(attempt int) time.Duration
}

// ExponentialBackoffStrategy implements exponential backoff recovery.
type ExponentialBackoffStrategy struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxAttempts int
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy.
func NewExponentialBackoffStrategy() *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		MaxAttempts: 5,
	}
}

// CanRecover checks if the error is recoverable.
func (s *ExponentialBackoffStrategy) CanRecover(err error) bool {
	// Don't recover from validation errors
	if IsValidationError(err) {
		return false
	}

	// Service errors are usually recoverable
	if e, ok := err.(*OrchestratorError); ok {
		return e.Type == ErrorTypeService || e.Type == ErrorTypeInternal
	}

	return false
}

// Recover attempts to recover from the error.
func (s *ExponentialBackoffStrategy) Recover(ctx context.Context, err error) error {
	// Log the recovery attempt
	log.Warn().
		Err(err).
		Msg("Attempting error recovery")

	// In a real implementation, this would:
	// - Reset connections
	// - Clear caches
	// - Reinitialize services
	// - Retry the operation

	return nil
}

// GetBackoffDuration calculates the backoff duration for an attempt.
func (s *ExponentialBackoffStrategy) GetBackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return s.BaseDelay
	}

	// Calculate exponential backoff: base * 2^(attempt-1)
	delay := s.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > s.MaxDelay {
			return s.MaxDelay
		}
	}

	return delay
}

// RecoveryManager coordinates error recovery strategies.
type RecoveryManager struct {
	strategies []RecoveryStrategy
	attempts   map[string]int
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager() *RecoveryManager {
	return &RecoveryManager{
		strategies: []RecoveryStrategy{
			NewExponentialBackoffStrategy(),
		},
		attempts: make(map[string]int),
	}
}

// HandleError attempts to recover from an error.
func (m *RecoveryManager) HandleError(ctx context.Context, err error, operationID string) error {
	// Track attempts
	m.attempts[operationID]++

	// Find a suitable recovery strategy
	for _, strategy := range m.strategies {
		if strategy.CanRecover(err) {
			// Check if we've exceeded max attempts
			if backoff, ok := strategy.(*ExponentialBackoffStrategy); ok {
				if m.attempts[operationID] > backoff.MaxAttempts {
					return fmt.Errorf("max recovery attempts exceeded: %w", err)
				}
			}

			// Wait before recovery
			backoffDuration := strategy.GetBackoffDuration(m.attempts[operationID])
			log.Info().
				Dur("backoff", backoffDuration).
				Int("attempt", m.attempts[operationID]).
				Msg("Waiting before recovery attempt")

			select {
			case <-time.After(backoffDuration):
				// Attempt recovery
				if recoveryErr := strategy.Recover(ctx, err); recoveryErr != nil {
					return fmt.Errorf("recovery failed: %w", recoveryErr)
				}
				return nil

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// No recovery strategy available
	return err
}

// ResetAttempts clears the attempt counter for an operation.
func (m *RecoveryManager) ResetAttempts(operationID string) {
	delete(m.attempts, operationID)
}

// GetRecoveryHint provides user-friendly recovery suggestions.
func GetRecoveryHint(err error) string {
	if e, ok := err.(*OrchestratorError); ok {
		if e.Hint != "" {
			return e.Hint
		}

		// Provide default hints based on error type
		switch e.Type {
		case ErrorTypeValidation:
			return "Review the input parameters and ensure they meet the documented requirements"
		case ErrorTypeNotFound:
			return "Verify the resource exists and you have the correct identifier"
		case ErrorTypeState:
			return "Check the current workflow state and ensure the requested operation is valid"
		case ErrorTypeSession:
			return "The session may have expired. Start a new documentation session"
		case ErrorTypeService:
			return "An external service is unavailable. Check connectivity and retry"
		case ErrorTypeInternal:
			return "An unexpected error occurred. Check the logs and contact support if the issue persists"
		default:
			return "An error occurred. Check the error details and logs for more information"
		}
	}

	return "An unknown error occurred. Check the logs for details"
}
