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

// [START storage_remove_bucket_iam_member]
import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// removeBucketIamMember removes the bucket IAM member.
func removeBucketIamMember(bucketName string) error {
	// bucketName := "bucket-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return fmt.Errorf("Bucket(%q).IAM().Policy: %v", bucketName, err)
	}
	// Other valid prefixes are "serviceAccount:", "user:"
	// See the documentation for more values.
	// https://cloud.google.com/storage/docs/access-control/iam
	policy.Remove("group:cloud-logs@google.com", "roles/storage.objectViewer")
	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("Bucket(%q).IAM().SetPolicy: %v", bucketName, err)
	}
	// NOTE: It may be necessary to retry this operation if IAM policies are
	// being modified concurrently. SetPolicy will return an error if the policy
	// was modified since it was retrieved.
	return nil
}

// [END storage_remove_bucket_iam_member]
