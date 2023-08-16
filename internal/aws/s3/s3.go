//go:build !func_test

package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/ptr"

	"github.com/elishacatherasoo/rain/internal/aws"
	"github.com/elishacatherasoo/rain/internal/aws/sts"
	"github.com/elishacatherasoo/rain/internal/config"
	"github.com/elishacatherasoo/rain/internal/console"
	"github.com/elishacatherasoo/rain/internal/console/spinner"
)

var BucketName = ""
var BucketKeyPrefix = ""

func getClient() *s3.Client {
	return s3.NewFromConfig(aws.Config())
}

// Returns true if the bucket is not empty
func BucketHasContents(bucketName string) (bool, error) {

	res, err := getClient().ListObjectVersions(context.Background(),
		&s3.ListObjectVersionsInput{
			Bucket: ptr.String(bucketName),
		})
	if err != nil {
		return false, err
	}
	if res.Versions != nil && len(res.Versions) > 0 {
		return true, nil
	}
	return false, nil
}

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) (bool, error) {
	_, err := getClient().HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: ptr.String(bucketName),
	})

	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: ptr.String(bucketName),
		ACL:    types.BucketCannedACLPrivate,
	}

	// We need a location constraint everywhere except us-east-1
	if region := aws.Config().Region; region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := getClient().CreateBucket(context.Background(), input)
	if err != nil {
		return err
	}

	// Encrypt the bucket
	_, err = getClient().PutBucketEncryption(context.Background(), &s3.PutBucketEncryptionInput{
		Bucket: ptr.String(bucketName),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAes256,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// Add public access block
	_, err = getClient().PutPublicAccessBlock(context.Background(), &s3.PutPublicAccessBlockInput{
		Bucket: ptr.String(bucketName),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       true,
			BlockPublicPolicy:     true,
			IgnorePublicAcls:      true,
			RestrictPublicBuckets: true,
		},
	})
	if err != nil {
		return err
	}

	// Add lifecycle config
	_, err = getClient().PutBucketLifecycleConfiguration(context.Background(), &s3.PutBucketLifecycleConfigurationInput{
		Bucket: ptr.String(bucketName),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					Status: types.ExpirationStatusEnabled,
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
						DaysAfterInitiation: 7,
					},
					Expiration: &types.LifecycleExpiration{
						Days: 7,
					},
					Filter: &types.LifecycleRuleFilterMemberPrefix{
						Value: "",
					},
					ID: ptr.String("delete after 14 days"),
					NoncurrentVersionExpiration: &types.NoncurrentVersionExpiration{
						NoncurrentDays: 7,
					},
				},
			},
		},
	})

	return err
}

// Upload an artefact to the bucket with a unique name
func Upload(bucketName string, content []byte) (string, error) {
	isBucketExists, errBucketExists := BucketExists(bucketName)

	if errBucketExists != nil {
		return "", fmt.Errorf("unable to confirm whether artifact bucket exists: %w", errBucketExists)
	}

	if !isBucketExists {
		return "", fmt.Errorf("bucket does not exist: '%s'", bucketName)
	}

	key := filepath.Join(BucketKeyPrefix, fmt.Sprintf("%x", sha256.Sum256(content)))

	_, err := getClient().PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: ptr.String(bucketName),
		Key:    ptr.String(key),
		ACL:    types.ObjectCannedACLPrivate,
		Body:   bytes.NewReader(content),
	})

	config.Debugf("Artifact key: %s", key)

	return key, err
}

// RainBucket returns the name of the rain deployment bucket in the current region
// and asks the user if they wish it to be created if it does not exist
// unless forceCreation is true, then it will not ask
func RainBucket(forceCreation bool) string {
	accountID, err := sts.GetAccountID()
	if err != nil {
		panic(fmt.Errorf("unable to get account ID: %w", err))
	}

	bucketName := BucketName
	if bucketName == "" {
		bucketName = fmt.Sprintf("rain-artifacts-%s-%s", accountID, aws.Config().Region)
	}

	config.Debugf("Artifact bucket: %s", bucketName)

	isBucketExists, err := BucketExists(bucketName)
	if err != nil {
		panic(fmt.Errorf("unable to confirm whether artifact bucket exists: %w", err))
	}

	if !isBucketExists {
		spinner.Pause()
		if !forceCreation && !console.Confirm(true, fmt.Sprintf("Rain needs to create an S3 bucket called '%s'. Continue?", bucketName)) {
			panic(errors.New("you may create the bucket manually and then re-run this operation"))
		}
		spinner.Resume()

		err := CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to create artifact bucket '%s': %w", bucketName, err))
		}
	}

	return bucketName
}
