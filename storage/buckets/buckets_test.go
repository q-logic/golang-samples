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

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/golang-samples/internal/testutil"
)

func TestMain(m *testing.M) {
	s := m.Run()
	os.Exit(s)
}

func TestCreate(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	// Clean up bucket before running tests.
	deleteBucket(bucketName)
	if err := create(tc.ProjectID, bucketName); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestCreateWithAttrs(t *testing.T) {
	tc := testutil.SystemTest(t)
	name := tc.ProjectID + "-storage-buckets-tests-attrs"

	// Clean up bucket before running test.
	deleteBucket(name)
	if err := createWithAttrs(tc.ProjectID, name); err != nil {
		t.Fatalf("createWithAttrs: %v", err)
	}
	if err := deleteBucket(name); err != nil {
		t.Fatalf("deleteBucket: %v", err)
	}
}

func TestList(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	buckets, err := list(tc.ProjectID)
	if err != nil {
		t.Fatal(err)
	}

	var ok bool
outer:
	for attempt := 0; attempt < 5; attempt++ { // for eventual consistency
		for _, b := range buckets {
			if b == bucketName {
				ok = true
				break outer
			}
		}
		time.Sleep(2 * time.Second)
	}
	if !ok {
		t.Errorf("got bucket list: %v; want %q in the list", buckets, bucketName)
	}
}

func TestGetBucketMetadata(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	buf := new(bytes.Buffer)
	if _, err := getBucketMetadata(buf, bucketName); err != nil {
		t.Errorf("getBucketMetadata: %#v", err)
	}

	got := buf.String()
	if want := "BucketName:"; !strings.Contains(got, want) {
		t.Errorf("contents = %q, want %q", got, want)
	}
}

func TestIAM(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	if _, err := getPolicy(ioutil.Discard, bucketName); err != nil {
		t.Errorf("getPolicy: %#v", err)
	}
	if err := addUser(bucketName); err != nil {
		t.Errorf("addUser: %v", err)
	}
	if err := removeUser(bucketName); err != nil {
		t.Errorf("removeUser: %v", err)
	}
}

func TestRequesterPays(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	if err := enableRequesterPays(bucketName); err != nil {
		t.Errorf("enableRequesterPays: %#v", err)
	}
	if err := disableRequesterPays(bucketName); err != nil {
		t.Errorf("disableRequesterPays: %#v", err)
	}
	if err := checkRequesterPays(ioutil.Discard, bucketName); err != nil {
		t.Errorf("checkRequesterPays: %#v", err)
	}
}

func TestKMS(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	keyRingID := os.Getenv("GOLANG_SAMPLES_KMS_KEYRING")
	cryptoKeyID := os.Getenv("GOLANG_SAMPLES_KMS_CRYPTOKEY")

	if keyRingID == "" || cryptoKeyID == "" {
		t.Skip("GOLANG_SAMPLES_KMS_KEYRING and GOLANG_SAMPLES_KMS_CRYPTOKEY must be set")
	}

	kmsKeyName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", tc.ProjectID, "global", keyRingID, cryptoKeyID)
	if err := setDefaultKMSkey(bucketName, kmsKeyName); err != nil {
		t.Fatalf("setDefaultKMSkey: failed to enable default kms key (%q): %v", kmsKeyName, err)
	}
}

func TestBucketLock(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	retentionPeriod := 5 * time.Second
	if err := setRetentionPolicy(bucketName, retentionPeriod); err != nil {
		t.Fatalf("setRetentionPolicy: %v", err)
	}

	attrs, err := getRetentionPolicy(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getRetentionPolicy: %v", err)
	}
	if attrs.RetentionPolicy.RetentionPeriod != retentionPeriod {
		t.Fatalf("retention period is not the expected value (%q): %v", retentionPeriod, attrs.RetentionPolicy.RetentionPeriod)
	}
	if err := enableDefaultEventBasedHold(bucketName); err != nil {
		t.Fatalf("enableDefaultEventBasedHold: %v", err)
	}

	attrs, err = getDefaultEventBasedHold(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getDefaultEventBasedHold: %v", err)
	}
	if !attrs.DefaultEventBasedHold {
		t.Fatalf("default event-based hold was not enabled")
	}
	if err := disableDefaultEventBasedHold(bucketName); err != nil {
		t.Fatalf("disableDefaultEventBasedHold: %v", err)
	}

	attrs, err = getDefaultEventBasedHold(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getDefaultEventBasedHold: %v", err)
	}
	if attrs.DefaultEventBasedHold {
		t.Fatalf("default event-based hold was not disabled")
	}
	if err := removeRetentionPolicy(bucketName); err != nil {
		t.Fatalf("removeRetentionPolicy: %v", err)
	}

	attrs, err = getRetentionPolicy(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getRetentionPolicy: %v", err)
	}
	if attrs.RetentionPolicy != nil {
		t.Fatalf("retention period to not be set")
	}
	if err := setRetentionPolicy(bucketName, retentionPeriod); err != nil {
		t.Fatalf("setRetentionPolicy: %v", err)
	}

	testutil.Retry(t, 10, time.Second, func(r *testutil.R) {
		if err := lockRetentionPolicy(ioutil.Discard, bucketName); err != nil {
			r.Errorf("lockRetentionPolicy: %v", err)
		}
		attrs, err := getRetentionPolicy(ioutil.Discard, bucketName)
		if err != nil {
			r.Errorf("getRetentionPolicy: %v", err)
		}
		if !attrs.RetentionPolicy.IsLocked {
			r.Errorf("retention policy is not locked")
		}
	})

	time.Sleep(5 * time.Second)
	deleteBucket(bucketName)
	time.Sleep(5 * time.Second)

	if err := create(tc.ProjectID, bucketName); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestUniformBucketLevelAccess(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	if err := enableUniformBucketLevelAccess(bucketName); err != nil {
		t.Fatalf("enableUniformBucketLevelAccess: %v", err)
	}

	attrs, err := getUniformBucketLevelAccess(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getUniformBucketLevelAccess: %v", err)
	}
	if !attrs.UniformBucketLevelAccess.Enabled {
		t.Fatalf("Uniform bucket-level access was not enabled for (%q).", bucketName)
	}
	if err := disableUniformBucketLevelAccess(bucketName); err != nil {
		t.Fatalf("disableUniformBucketLevelAccess: %v", err)
	}

	attrs, err = getUniformBucketLevelAccess(ioutil.Discard, bucketName)
	if err != nil {
		t.Fatalf("getUniformBucketLevelAccess: %v", err)
	}
	if attrs.UniformBucketLevelAccess.Enabled {
		t.Fatalf("Uniform bucket-level access was not disabled for (%q).", bucketName)
	}
}

func TestDelete(t *testing.T) {
	tc := testutil.SystemTest(t)
	bucketName := tc.ProjectID + "-storage-buckets-tests"

	if err := deleteBucket(bucketName); err != nil {
		t.Fatalf("deleteBucket: %v", err)
	}
}
