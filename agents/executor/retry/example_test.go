/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chainguard.dev/driftlessaf/agents/executor/retry"
)

// ExampleRetryWithBackoff demonstrates using RetryWithBackoff to handle
// transient API errors with exponential backoff.
func ExampleRetryWithBackoff() {
	ctx := context.Background()
	cfg := retry.DefaultRetryConfig()

	// Define what errors are retryable
	isRetryable := func(err error) bool {
		return errors.Is(err, errRateLimited)
	}

	// Simulate an API call that may fail transiently
	result, err := retry.RetryWithBackoff(
		ctx,
		cfg,
		"fetch_user",
		isRetryable,
		func() (string, error) {
			// Your API call here
			return "user-data", nil
		},
	)
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		return
	}
	fmt.Printf("success: %s\n", result)
	// Output: success: user-data
}

// ExampleRetryWithBackoff_customConfig demonstrates using a custom RetryConfig
// for specific retry requirements.
func ExampleRetryWithBackoff_customConfig() {
	ctx := context.Background()

	// Custom configuration for aggressive retries
	cfg := retry.RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 100 * time.Millisecond,
		MaxBackoff:  5 * time.Second,
		MaxJitter:   100 * time.Millisecond,
	}

	isRetryable := func(err error) bool {
		return errors.Is(err, errRateLimited)
	}

	result, err := retry.RetryWithBackoff(
		ctx,
		cfg,
		"quick_operation",
		isRetryable,
		func() (int, error) {
			return 42, nil
		},
	)
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		return
	}
	fmt.Printf("result: %d\n", result)
	// Output: result: 42
}

// ExampleDefaultRetryConfig demonstrates getting the default retry configuration.
func ExampleDefaultRetryConfig() {
	cfg := retry.DefaultRetryConfig()

	fmt.Printf("MaxRetries: %d\n", cfg.MaxRetries)
	fmt.Printf("BaseBackoff: %v\n", cfg.BaseBackoff)
	fmt.Printf("MaxBackoff: %v\n", cfg.MaxBackoff)
	fmt.Printf("MaxJitter: %v\n", cfg.MaxJitter)
	// Output:
	// MaxRetries: 5
	// BaseBackoff: 1s
	// MaxBackoff: 1m0s
	// MaxJitter: 500ms
}

// ExampleRetryConfig_Validate demonstrates validating a retry configuration.
func ExampleRetryConfig_Validate() {
	// Valid configuration
	validCfg := retry.RetryConfig{
		MaxRetries:  5,
		BaseBackoff: time.Second,
		MaxBackoff:  time.Minute,
		MaxJitter:   500 * time.Millisecond,
	}
	if err := validCfg.Validate(); err != nil {
		fmt.Printf("validation failed: %v\n", err)
	} else {
		fmt.Println("valid configuration")
	}

	// Invalid configuration with negative values
	invalidCfg := retry.RetryConfig{
		MaxRetries:  -1,
		BaseBackoff: time.Second,
		MaxBackoff:  time.Minute,
		MaxJitter:   500 * time.Millisecond,
	}
	if err := invalidCfg.Validate(); err != nil {
		fmt.Printf("validation failed: %v\n", err)
	}
	// Output:
	// valid configuration
	// validation failed: max retries cannot be negative
}

// ExampleRequeueIfRetryable demonstrates using RequeueIfRetryable to convert
// exhausted retries into a workqueue requeue.
func ExampleRequeueIfRetryable() {
	ctx := context.Background()

	// Simulate an error after retries are exhausted
	apiErr := errors.New("429 rate limit exceeded")

	isRetryable := func(err error) bool {
		return errors.Is(err, apiErr)
	}

	// Check if the error should trigger a requeue
	requeueErr := retry.RequeueIfRetryable(ctx, apiErr, isRetryable, "OpenAI")
	if requeueErr != nil {
		fmt.Println("requeue requested")
		return
	}
	fmt.Println("no requeue needed")
	// Output: requeue requested
}

// ExampleRequeueIfRetryable_nonRetryable demonstrates that non-retryable errors
// do not trigger a requeue.
func ExampleRequeueIfRetryable_nonRetryable() {
	ctx := context.Background()

	// Permanent error that should not be retried
	permErr := errors.New("permission denied")

	isRetryable := func(err error) bool {
		return false // Not retryable
	}

	requeueErr := retry.RequeueIfRetryable(ctx, permErr, isRetryable, "API")
	if requeueErr != nil {
		fmt.Println("requeue requested")
		return
	}
	fmt.Println("no requeue needed")
	// Output: no requeue needed
}

// errRateLimited is a sentinel error for examples.
var errRateLimited = errors.New("rate limited")
