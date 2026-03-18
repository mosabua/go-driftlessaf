/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package apkurl provides utilities for parsing APK URL keys used by the
// apkreconciler workqueue.
//
// APK URL keys are of the form "{host}/{repo-path...}/{arch}/{package}-{version}.apk"
// and do not include the scheme (https://). Use Parse to decode a key into its
// structured components.
package apkurl
