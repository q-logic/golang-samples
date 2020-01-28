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

// [START storage_lock_retention_policy]
import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// lockRetentionPolicy locks bucket retention policy.
func lockRetentionPolicy(w io.Writer, bucketName string) error {
	// bucketName := "bucket-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	attrs, err := client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return err
	}

	conditions := storage.BucketConditions{
		MetagenerationMatch: attrs.MetaGeneration,
	}
	if err := bucket.If(conditions).LockRetentionPolicy(ctx); err != nil {
		return err
	}

	lockedAttrs, err := client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Retention policy for %v is now locked\n", bucketName)
	fmt.Fprintf(w, "Retention policy effective as of %v\n", lockedAttrs.RetentionPolicy.EffectiveTime)
	return nil
}

// [END storage_lock_retention_policy]
