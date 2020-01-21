// Copyright 2019 Google LLC
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
package acl

// [START bucket_list_acl_filtered]
import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// bucketListACLFiltered lists bucket ACL using a filter.
func bucketListACLFiltered(bucket string, entity storage.ACLEntity) error {
	// bucket := "bucket-name"
	// entity := storage.AllAuthenticatedUsers
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	rules, err := client.Bucket(bucket).ACL().List(ctx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		if r.Entity == entity {
			fmt.Printf("ACL rule role: %v\n", r.Role)
		}
	}
	return nil
}

// [END bucket_list_acl_filtered]
