package advanced

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ProcessData performs data processing with retry logic
func ProcessData(ctx context.Context, data string) (string, error) {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := callAPI(ctx, data)
		if err == nil {
			return result, nil
		}
		if attempt < maxRetries-1 {
			backoff := calculateBackoff(attempt)
			time.Sleep(backoff)
		}
	}
	return "", errors.New("all retries exhausted")
}

// calculateBackoff returns exponential backoff duration
func calculateBackoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

// callAPI simulates an external API call
func callAPI(ctx context.Context, data string) (string, error) {
	// Simulate network call
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return fmt.Sprintf("processed: %s", data), nil
	}
}

// DataPipeline processes data through multiple stages
func DataPipeline(ctx context.Context, input string) (string, error) {
	// Stage 1: Transform
	transformed, err := transformData(ctx, input)
	if err != nil {
		return "", fmt.Errorf("transform failed: %w", err)
	}

	// Stage 2: Enrich
	enriched, err := enrichData(ctx, transformed)
	if err != nil {
		return "", fmt.Errorf("enrich failed: %w", err)
	}

	// Stage 3: Finalize
	result, err := finalizeData(ctx, enriched)
	if err != nil {
		return "", fmt.Errorf("finalize failed: %w", err)
	}

	return result, nil
}

// transformData performs the transformation stage
func transformData(ctx context.Context, input string) (string, error) {
	if input == "" {
		return "", errors.New("empty input")
	}
	return fmt.Sprintf("transformed[%s]", input), nil
}

// enrichData adds additional information to the data
func enrichData(ctx context.Context, data string) (string, error) {
	// Check if enrichment data is available in context
	if enrichmentData := ctx.Value("enrichment"); enrichmentData != nil {
		return fmt.Sprintf("%s+%v", data, enrichmentData), nil
	}
	return fmt.Sprintf("%s+default", data), nil
}

// finalizeData completes the processing pipeline
func finalizeData(ctx context.Context, data string) (string, error) {
	return fmt.Sprintf("final{%s}", data), nil
}

// State represents the state machine states
type State int

const (
	StateIdle State = iota
	StateProcessing
	StateValidating
	StateCompleted
	StateError
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateProcessing:
		return "Processing"
	case StateValidating:
		return "Validating"
	case StateCompleted:
		return "Completed"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// StateMachine manages state transitions
type StateMachine struct {
	currentState State
	data         string
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState: StateIdle,
	}
}

// Transition moves the state machine through its lifecycle
func (sm *StateMachine) Transition(ctx context.Context, event string) error {
	nextState := computeNextState(sm.currentState, event)

	// Validate transition with business rules
	if !checkRule(ctx, sm.currentState, nextState) {
		return fmt.Errorf("invalid transition from %v to %v", sm.currentState, nextState)
	}

	sm.currentState = nextState
	return nil
}

// GetState returns the current state
func (sm *StateMachine) GetState() State {
	return sm.currentState
}

// computeNextState determines the next state based on current state and event
func computeNextState(current State, event string) State {
	switch current {
	case StateIdle:
		if event == "start" {
			return StateProcessing
		}
	case StateProcessing:
		if event == "validate" {
			return StateValidating
		} else if event == "error" {
			return StateError
		}
	case StateValidating:
		if event == "complete" {
			return StateCompleted
		} else if event == "error" {
			return StateError
		}
	case StateError:
		if event == "retry" {
			return StateIdle
		}
	}
	return current // Stay in current state if no valid transition
}

// checkRule validates if a state transition is allowed
func checkRule(ctx context.Context, from, to State) bool {
	// Check context for custom rules
	if rules := ctx.Value("rules"); rules != nil {
		if ruleMap, ok := rules.(map[string]bool); ok {
			key := fmt.Sprintf("%d->%d", from, to)
			if allowed, exists := ruleMap[key]; exists {
				return allowed
			}
		}
	}

	// Default rules: only allow forward progression or error/retry
	switch from {
	case StateIdle:
		return to == StateProcessing
	case StateProcessing:
		return to == StateValidating || to == StateError
	case StateValidating:
		return to == StateCompleted || to == StateError
	case StateError:
		return to == StateIdle
	case StateCompleted:
		return false // Terminal state
	}
	return false
}

// RunWorkflow executes a complete workflow with multiple function calls
func RunWorkflow(ctx context.Context, input string) (string, error) {
	// Initialize state machine
	sm := NewStateMachine()

	// Transition: Idle -> Processing
	if err := sm.Transition(ctx, "start"); err != nil {
		return "", err
	}

	// Process the data
	result, err := ProcessData(ctx, input)
	if err != nil {
		sm.Transition(ctx, "error")
		return "", err
	}

	// Transition: Processing -> Validating
	if err := sm.Transition(ctx, "validate"); err != nil {
		return "", err
	}

	// Run through pipeline
	pipelineResult, err := DataPipeline(ctx, result)
	if err != nil {
		sm.Transition(ctx, "error")
		return "", err
	}

	// Transition: Validating -> Completed
	if err := sm.Transition(ctx, "complete"); err != nil {
		return "", err
	}

	return pipelineResult, nil
}
