/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gitsign provides a git commit signer backed by Sigstore keyless
// signing.
//
// It uses Fulcio for certificate issuance and Rekor for transparency log
// inclusion, obtaining an OIDC token from the ambient Google Cloud credential
// provider (e.g., a Cloud Run service account).
package gitsign
