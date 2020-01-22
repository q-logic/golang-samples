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
package objects

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/storage"

	"github.com/GoogleCloudPlatform/golang-samples/internal/testutil"
)

func TestMain(m *testing.M) {
	// These functions are noisy.
	log.SetOutput(ioutil.Discard)
	s := m.Run()
	log.SetOutput(os.Stderr)
	os.Exit(s)
}

// TestObjects runs all samples tests of the package.
func TestObjects(t *testing.T) {
	tc := testutil.SystemTest(t)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}

	var (
		bucket    = tc.ProjectID + "-samples-object-bucket-1"
		dstBucket = tc.ProjectID + "-samples-object-bucket-2"

		object1 = "foo.txt"
		object2 = "foo/a.txt"
	)

	cleanBucket(t, ctx, client, tc.ProjectID, bucket)
	cleanBucket(t, ctx, client, tc.ProjectID, dstBucket)

	if err := write(bucket, object1); err != nil {
		t.Fatalf("write(%q): %v", object1, err)
	}
	if err := write(bucket, object2); err != nil {
		t.Fatalf("write(%q): %v", object2, err)
	}

	{
		// Should only show "foo/a.txt", not "foo.txt"
		var buf bytes.Buffer
		if err := list(&buf, bucket); err != nil {
			t.Fatalf("cannot list objects: %v", err)
		}
		if got, want := buf.String(), object1; !strings.Contains(got, want) {
			t.Errorf("List() got %q; want to contain %q", got, want)
		}
		if got, want := buf.String(), object2; !strings.Contains(got, want) {
			t.Errorf("List() got %q; want to contain %q", got, want)
		}
	}

	{
		// Should only show "foo/a.txt", not "foo.txt"
		const prefix = "foo/"
		var buf bytes.Buffer
		if err := listByPrefix(&buf, bucket, prefix, ""); err != nil {
			t.Fatalf("cannot list objects by prefix: %v", err)
		}
		if got, want := buf.String(), object1; strings.Contains(got, want) {
			t.Errorf("List(%q) got %q; want NOT to contain %q", prefix, got, want)
		}
		if got, want := buf.String(), object2; !strings.Contains(got, want) {
			t.Errorf("List(%q) got %q; want to contain %q", prefix, got, want)
		}
	}

	{
		var buf bytes.Buffer
		if err := downloadUsingRequesterPays(&buf, bucket, object1, tc.ProjectID); err != nil {
			t.Errorf("cannot download using requester pays: %v", err)
		}
	}

	data, err := read(bucket, object1)
	if err != nil {
		t.Fatalf("cannot read object: %v", err)
	}
	if got, want := string(data), "Hello\nworld"; got != want {
		t.Errorf("contents = %q; want %q", got, want)
	}
	_, err = attrs(bucket, object1)
	if err != nil {
		t.Errorf("cannot get object metadata: %v", err)
	}
	if err := makePublic(bucket, object1); err != nil {
		t.Errorf("cannot to make object public: %v", err)
	}
	err = move(bucket, object1)
	if err != nil {
		t.Fatalf("cannot move object: %v", err)
	}
	// object1's new name.
	object1 = object1 + "-rename"

	if err := copyToBucket(dstBucket, bucket, object1); err != nil {
		t.Errorf("cannot copy object to bucket: %v", err)
	}

	key := []byte("my-secret-AES-256-encryption-key")
	newKey := []byte("My-secret-AES-256-encryption-key")

	if err := writeEncryptedObject(bucket, object1, key); err != nil {
		t.Errorf("cannot write an encrypted object: %v", err)
	}
	data, err = readEncryptedObject(bucket, object1, key)
	if err != nil {
		t.Errorf("cannot read the encrypted object: %v", err)
	}
	if got, want := string(data), "top secret"; got != want {
		t.Errorf("object content = %q; want %q", got, want)
	}
	if err := rotateEncryptionKey(bucket, object1, key, newKey); err != nil {
		t.Errorf("cannot encrypt the object with the new key: %v", err)
	}
	if err := delete(bucket, object1); err != nil {
		t.Errorf("cannot to delete object: %v", err)
	}
	if err := delete(bucket, object2); err != nil {
		t.Errorf("cannot to delete object: %v", err)
	}

	testutil.Retry(t, 10, time.Second, func(r *testutil.R) {
		// Cleanup, this part won't be executed if Fatal happens.
		// TODO(jbd): Implement garbage cleaning.
		if err := client.Bucket(bucket).Delete(ctx); err != nil {
			r.Errorf("cleanup of bucket failed: %v", err)
		}
	})

	testutil.Retry(t, 10, time.Second, func(r *testutil.R) {
		if err := delete(dstBucket, object1+"-copy"); err != nil {
			r.Errorf("cannot to delete copy object: %v", err)
		}
	})

	testutil.Retry(t, 10, time.Second, func(r *testutil.R) {
		if err := client.Bucket(dstBucket).Delete(ctx); err != nil {
			r.Errorf("cleanup of bucket failed: %v", err)
		}
	})
}

func TestKMSObjects(t *testing.T) {
	tc := testutil.SystemTest(t)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}

	keyRingID := os.Getenv("GOLANG_SAMPLES_KMS_KEYRING")
	cryptoKeyID := os.Getenv("GOLANG_SAMPLES_KMS_CRYPTOKEY")
	if keyRingID == "" || cryptoKeyID == "" {
		t.Skip("GOLANG_SAMPLES_KMS_KEYRING and GOLANG_SAMPLES_KMS_CRYPTOKEY must be set")
	}

	var (
		bucket    = tc.ProjectID + "-samples-object-bucket-1"
		dstBucket = tc.ProjectID + "-samples-object-bucket-2"

		object1 = "foo.txt"
	)

	cleanBucket(t, ctx, client, tc.ProjectID, bucket)
	cleanBucket(t, ctx, client, tc.ProjectID, dstBucket)

	kmsKeyName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", tc.ProjectID, "global", keyRingID, cryptoKeyID)

	if err := writeWithKMSKey(bucket, object1, kmsKeyName); err != nil {
		t.Errorf("cannot write a KMS encrypted object: %v", err)
	}
}

func TestV4SignedURL(t *testing.T) {
	tc := testutil.SystemTest(t)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}

	bucketName := tc.ProjectID + "-signed-url-bucket-name"
	objectName := "foo.txt"
	serviceAccount := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	cleanBucket(t, ctx, client, tc.ProjectID, bucketName)
	putBuf := new(bytes.Buffer)
	putURL, err := generateV4PutObjectSignedURL(putBuf, bucketName, objectName, serviceAccount)
	if err != nil {
		t.Errorf("generateV4PutObjectSignedURL: %v", err)
	}
	got := putBuf.String()
	if want := "Generated PUT signed URL:"; !strings.Contains(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	httpClient := &http.Client{}
	request, err := http.NewRequest("PUT", putURL, strings.NewReader("hello world"))
	request.ContentLength = 11
	request.Header.Set("Content-Type", "application/octet-stream")
	response, err := httpClient.Do(request)
	if err != nil {
		t.Errorf("httpClient.Do: %v", err)
	}
	getBuf := new(bytes.Buffer)
	getURL, err := generateV4GetObjectSignedURL(getBuf, bucketName, objectName, serviceAccount)
	if err != nil {
		t.Errorf("generateV4GetObjectSignedURL: %v", err)
	}
	got = getBuf.String()
	if want := "Generated GET signed URL:"; !strings.Contains(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	response, err = http.Get(getURL)
	if err != nil {
		t.Errorf("http.Get: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("ioutil.ReadAll: %v", err)
	}

	if got, want := string(body), "hello world"; got != want {
		t.Errorf("object content = %q; want %q", got, want)
	}

}

func TestObjectBucketLock(t *testing.T) {
	tc := testutil.SystemTest(t)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}

	var (
		bucketName = tc.ProjectID + "-retent-samples-object-bucket"

		objectName = "foo.txt"

		retentionPeriod = 5 * time.Second
	)

	cleanBucket(t, ctx, client, tc.ProjectID, bucketName)
	bucket := client.Bucket(bucketName)

	if err := write(bucketName, objectName); err != nil {
		t.Fatalf("write(%q): %v", objectName, err)
	}
	if _, err := bucket.Update(ctx, storage.BucketAttrsToUpdate{
		RetentionPolicy: &storage.RetentionPolicy{
			RetentionPeriod: retentionPeriod,
		},
	}); err != nil {
		t.Errorf("unable to set retention policy (%q): %v", bucketName, err)
	}
	if err := setEventBasedHold(bucketName, objectName); err != nil {
		t.Errorf("unable to set event-based hold (%q/%q): %v", bucketName, objectName, err)
	}
	oAttrs, err := attrs(bucketName, objectName)
	if err != nil {
		t.Errorf("cannot get object metadata: %v", err)
	}
	if !oAttrs.EventBasedHold {
		t.Errorf("event-based hold is not enabled")
	}
	if err := releaseEventBasedHold(bucketName, objectName); err != nil {
		t.Errorf("unable to set event-based hold (%q/%q): %v", bucketName, objectName, err)
	}
	oAttrs, err = attrs(bucketName, objectName)
	if err != nil {
		t.Errorf("cannot get object metadata: %v", err)
	}
	if oAttrs.EventBasedHold {
		t.Errorf("event-based hold is not disabled")
	}
	if _, err := bucket.Update(ctx, storage.BucketAttrsToUpdate{
		RetentionPolicy: &storage.RetentionPolicy{},
	}); err != nil {
		t.Errorf("unable to remove retention policy (%q): %v", bucketName, err)
	}
	if err := setTemporaryHold(bucketName, objectName); err != nil {
		t.Errorf("unable to set temporary hold (%q/%q): %v", bucketName, objectName, err)
	}
	oAttrs, err = attrs(bucketName, objectName)
	if err != nil {
		t.Errorf("cannot get object metadata: %v", err)
	}
	if !oAttrs.TemporaryHold {
		t.Errorf("temporary hold is not disabled")
	}
	if err := releaseTemporaryHold(bucketName, objectName); err != nil {
		t.Errorf("unable to release temporary hold (%q/%q): %v", bucketName, objectName, err)
	}
	oAttrs, err = attrs(bucketName, objectName)
	if err != nil {
		t.Errorf("cannot get object metadata: %v", err)
	}
	if oAttrs.TemporaryHold {
		t.Errorf("temporary hold is not disabled")
	}
}

// cleanBucket ensures there's a fresh bucket with a given name, deleting the existing bucket if it already exists.
func cleanBucket(t *testing.T, ctx context.Context, client *storage.Client, projectID, bucket string) {
	b := client.Bucket(bucket)
	_, err := b.Attrs(ctx)
	if err == nil {
		it := b.Objects(ctx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				t.Fatalf("Bucket.Objects(%q): %v", bucket, err)
			}
			if attrs.EventBasedHold || attrs.TemporaryHold {
				if _, err := b.Object(attrs.Name).Update(ctx, storage.ObjectAttrsToUpdate{
					TemporaryHold:  false,
					EventBasedHold: false,
				}); err != nil {
					t.Fatalf("Bucket(%q).Object(%q).Update: %v", bucket, attrs.Name, err)
				}
			}
			if err := b.Object(attrs.Name).Delete(ctx); err != nil {
				t.Fatalf("Bucket(%q).Object(%q).Delete: %v", bucket, attrs.Name, err)
			}
		}
		if err := b.Delete(ctx); err != nil {
			t.Fatalf("Bucket.Delete(%q): %v", bucket, err)
		}
	}
	if err := b.Create(ctx, projectID, nil); err != nil {
		t.Fatalf("Bucket.Create(%q): %v", bucket, err)
	}
}
