package s3client

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/matthew-andrews/s3up/objects"
	"os"
)

type s3Interface interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
	HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	CopyObject(*s3.CopyObjectInput) (*s3.CopyObjectOutput, error)
}

type client struct {
	Service s3Interface
}

func New(service s3Interface) client {
	return client{
		Service: service,
	}
}

func (client client) Upload(bucket string, files []objects.File) error {
	if len(files) < 1 {
		return errors.New("No files found for upload to S3.  (Directories are ignored)")
	}

	for _, file := range files {
		if err := client.uploadFile(bucket, file); err != nil {
			return err
		}
	}

	return nil
}

func (client client) uploadFile(bucket string, file objects.File) error {
	// HeadObject
	headResp, err := client.Service.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file.Key),
	})

	if err != nil {
		return errors.New(fmt.Sprintf("Head request to S3 object failed: %s", err))
	}

	if aws.StringValue(headResp.ETag) != "\""+file.ETag+"\"" {
		// PutObject if ETags mismatch
		realFile, err := os.Open(file.Location)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not open file: %s", file.Location))
		}
		defer realFile.Close()

		putObjectInput := &s3.PutObjectInput{
			Body:         realFile,
			Bucket:       aws.String(bucket),
			Key:          aws.String(file.Key),
			ContentType:  aws.String(file.ContentType),
			CacheControl: aws.String(file.CacheControl),
			ACL:          aws.String(file.ACL),
		}

		_, err = client.Service.PutObject(putObjectInput)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to update file: %s", err.Error()))
		}
		fmt.Printf("Successfully uploaded: %s\n", file.Key)

	} else if aws.StringValue(headResp.CacheControl) != file.CacheControl || aws.StringValue(headResp.ContentType) != file.ContentType {
		// CopyObject if ETags match but something else doesn't
		copyObjectInput := &s3.CopyObjectInput{
			Bucket:            aws.String(bucket),
			CopySource:        aws.String(bucket + "/" + file.Key),
			Key:               aws.String(file.Key),
			ContentType:       aws.String(file.ContentType),
			CacheControl:      aws.String(file.CacheControl),
			ACL:               aws.String(file.ACL),
			MetadataDirective: aws.String("REPLACE"),
		}

		_, err = client.Service.CopyObject(copyObjectInput)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to update file: %s", err.Error()))
		}
		fmt.Printf("Successfully updated just metadata of: %s\n", file.Key)
	} else {
		fmt.Printf("Unchanged, skipping: %s\n", file.Key)
	}

	return nil
}
