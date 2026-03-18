/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitsign_test

import (
	"fmt"
)

// Example_newSigner demonstrates the intended usage of NewSigner.
//
// NewSigner requires ambient Google Cloud credentials (e.g., a Cloud Run
// service account) and a Sigstore provider to be enabled. It is not
// suitable for unit tests without those prerequisites.
func Example_newSigner() {
	// In production, call gitsign.NewSigner(ctx) to obtain a git.Signer
	// backed by Sigstore keyless signing:
	//
	//   signer, err := gitsign.NewSigner(ctx)
	//   if err != nil {
	//       return fmt.Errorf("creating signer: %w", err)
	//   }
	//   // Pass signer to go-git commit options.
	fmt.Println("NewSigner requires ambient GCP credentials")
	// Output: NewSigner requires ambient GCP credentials
}
