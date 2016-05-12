package main

import (
	//	"errors"

	"bytes"
	"io"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

// UploadToS3 uploads a buffer to S3
func UploadToS3(svc s3.S3, destName, bucketName string, buffer []byte, size int64) {
	fileBytes := bytes.NewReader(buffer) // convert to io.ReadSeeker type

	fileType := http.DetectContentType(buffer)

	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),    // required
		Key:           aws.String(destName),      // required
		ACL:           aws.String("public-read"), // Needed to allow anonymous access
		Body:          fileBytes,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(fileType),
		Metadata: map[string]*string{
			"Key": aws.String("MetadataValue"), //required
		},
		// see more at http://godoc.org/github.com/aws/aws-sdk-go/service/s3#S3.PutObject
	}

	_, err := svc.PutObject(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// Generic AWS Error with Code, Message, and original error (if any)
			log.Error("AWS error: ", awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
			if reqErr, ok := err.(awserr.RequestFailure); ok {
				// A service error occurred
				log.Error("AWS service error: ", reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
			}
		} else {
			// This case should never be hit, the SDK should always return an
			// error which satisfies the awserr.Error interface.
			log.Fatal("Fatal AWS error: ", err.Error())
		}
	}
}

// GetFromS3 gets an object from S3
func GetFromS3(svc s3.S3, sourceName, bucketName string) (io.Reader, int64) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucketName), // required
		Key:    aws.String(sourceName), // required
		// see more at http://godoc.org/github.com/aws/aws-sdk-go/service/s3#S3.PutObject
	}

	//log.Info("Fetching ", sourceName, " from ", bucketName)
	resp, err := svc.GetObject(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchKey" {
				// file doesn exist
				return nil, 0
			}
			// Generic AWS Error with Code, Message, and original error (if any)
			log.Error("AWS error: ", awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
			if reqErr, ok := err.(awserr.RequestFailure); ok {
				// A service error occurred
				log.Error("AWS service error: ", reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
			}
		} else {
			// This case should never be hit, the SDK should always return an
			// error which satisfies the awserr.Error interface.
			log.Fatal("Fatal AWS error: ", err.Error())
		}
	}

	return resp.Body, *resp.ContentLength
}
