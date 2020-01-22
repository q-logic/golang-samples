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
package buckets

// [START get_requester_pays_status]
import (
	"context"
	"log"

	"cloud.google.com/go/storage"
)

// checkRequesterPays gets requester pays status.
func checkRequesterPays(bucketName string) error {
	// bucketName := "bucket-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	attrs, err := client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return err
	}
	log.Printf("Is requester pays enabled? %v\n", attrs.RequesterPays)
	return nil
}

// [END get_requester_pays_status]
