package main

import (
//	"errors"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Helper to log an error and then exit
func handleErr(err error) {
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

// Loops through all files in a dir
func getFilesInDir(dirName, outDir string) {
	files, err := ioutil.ReadDir(dirName)
	handleErr(err)

	for _, f := range files {
		fileName := dirName + "/" + f.Name()

		// Get date taken for file
		date, err := getDateTaken(fileName)
		if err != nil {
			log.Warn(err.Error())
		}

		// Organise photo by moving to target folder
		err = organisePhoto(dirName, f.Name(), outDir, date)
		if err != nil {
			log.Error(err.Error())
		}

		// Upload file to AWS S3 bucket
		err = uploadS3(fileName, date)
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func listBuckets() {
	svc := s3.New(session.New(&aws.Config{Region: aws.String("ap-southeast-2")}))
	result, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		log.Println("Failed to list buckets", err)
		return err
	}

	log.Println("Buckets:")
	for _, bucket := range result.Buckets {
		log.Printf("%s : %s\n", aws.StringValue(bucket.Name), bucket.CreationDate)
	}
	return nil
}

func uploadS3(fileName string, dateTaken time.Time) error {
	// TODO! Upload file to a S3 bucket
	svc := s3.New(session.New(&aws.Config{Region: aws.String("ap-southeast-2")}))
	result, err := svc. (&s3.ListBucketsInput{})
	if err != nil {
		log.Println("Failed to list buckets", err)
		return err
	}

	log.Println("Buckets:")
	for _, bucket := range result.Buckets {
		log.Printf("%s : %s\n", aws.StringValue(bucket.Name), bucket.CreationDate)
	}
	return nil
}

func main() {

	// Declare a string parameter
	inDirNamePtr := flag.String("in", ".", "input directory")
	bucketNamePtr := flag.String("bucket", "", "bucket name")
	// Parse command line arguments.
	flag.Parse()
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}
	if len(*bucketNamePtr) == 0 {
		log.Fatal("Error, need to define a bucket name.")
	}

	uploadS3(*inDirNamePtr, *bucketNamePtr)

}
