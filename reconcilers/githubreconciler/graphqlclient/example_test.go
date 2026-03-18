/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package graphqlclient_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/graphqlclient"
	"github.com/google/go-github/v75/github"
)

// ExampleNewGraphQLClient demonstrates creating a GraphQL client from a
// github.Client.
func ExampleNewGraphQLClient() {
	gh := github.NewClient(nil)
	client := graphqlclient.NewGraphQLClient(gh)
	fmt.Printf("client type: %T\n", client)
	// Output: client type: *graphqlclient.GraphQLClient
}
