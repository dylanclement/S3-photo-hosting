package main

import (
//	"errors"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	filePath "path/filepath"
	//"strings"
	"time"
)

// Loops through all files in a dir
func processDir(dirName, bucketName, outDir string) {
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
		err = processPhoto(fileName, bucketName, outDir, date)
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func listBuckets() error {
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

func uploadS3(fileName, bucketName, outPath string) error {
	// TODO! Upload file to a S3 bucket
	//svc := s3.New(session.New(&aws.Config{Region: aws.String("ap-southeast-2")}))

	log.Println("Uploading file to bucket: " + fileName)
	return nil
}


func processPhoto(fileName, bucketName, outDir string, dateTaken time.Time) error {
	outPath := filePath.Join(dateTaken.Format("2006/2006-01-02"), filePath.Base(fileName))
	if len(outDir) > 0 {
		destDir := filePath.Join(outDir, outPath)
		createDir(destDir)
		copyFile(fileName, destDir)
		log.Info("Copied file: " + fileName)
	}
	if len(bucketName) > 0 {
		uploadS3(fileName, bucketName, outPath)
		log.Info("Uploaded file to bucket" + bucketName)
	}
	// TODO! Write index.html file
	return nil
}


func main() {

	// Declare a string parameter
	inDirNamePtr := flag.String("i", ".", "input directory")
	outDirNamePtr := flag.String("o", "", "output directory")
	bucketNamePtr := flag.String("b", "", "bucket name")
	// Parse command line arguments.
	flag.Parse()
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}

	processDir(*inDirNamePtr, *bucketNamePtr, *outDirNamePtr)
}
