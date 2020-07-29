// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

// [START fs_listen_errors]
import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/firestore"
)

// listenErrors demonstrates how to handle listening errors.
func listenErrors(w io.Writer, projectID string) error {
	// projectID := "project-id"
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("firestore.NewClient: %v", err)
	}
	defer client.Close()

	qsnap := client.Collection("cities").Snapshots(ctx)
	snap, err := qsnap.Next()
	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	for _, change := range snap.Changes {
		if change.Kind == firestore.DocumentAdded {
			fmt.Fprintf(w, "New city: %v\n", change.Doc.Data())
		}
	}
	return nil
}

// [END fs_listen_errors]
