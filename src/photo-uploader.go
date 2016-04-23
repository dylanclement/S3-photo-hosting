package main

import (
	//	"errors"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	filepath "path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rwcarlsen/goexif/exif"
)

// Helper to log an error and then exit
// Helper to log an error and then exit
func handleErr(err error) {
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

// Helper to get file modification time, useful as a fallback if file is not a jpg.
func getFileModTime(fileName string) time.Time {
	stat, err := os.Stat(fileName)
	if err != nil {
		log.Error("Unable to get ModTime for file: ", fileName)
		return time.Now()
	}
	return stat.ModTime()
}

// Get date taken of a file. If it is a jpg it will attempt to use EXIF data
func getDateTaken(fileName string) (time.Time, error) {

	if len(fileName) <= 0 {
		log.Warn("Pass filename as parameter.")
		return time.Now(), errors.New("Invalid filename passed.")
	}

	file, err := os.Open(fileName)
	if err != nil {
		return time.Now(), err
	}

	fileExt := strings.ToLower(filepath.Ext(fileName))

	date := time.Now()

	if fileExt == ".jpg" {

		data, err := exif.Decode(file)
		if err != nil {
			// file might not have exif data, use os.Stat
			date = getFileModTime(fileName)
		} else {
			date, _ = data.DateTime()
		}
	} else {
		date = getFileModTime(fileName)
	}

	return date, err
}

// Helper to create a folder
func createDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// Ok directory doesn't exist, create it
		err := os.Mkdir(dirName, 0777)
		if err != nil {
			log.Warn("Error happened creating directory:", err.Error())
		}
	}
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func processPhoto(fileName, outDir, bucketName string, dateTaken time.Time) error {
	outPath := filepath.Join(dateTaken.Format("2006/2006-01-02"), filepath.Base(fileName))
	if len(outDir) > 0 {
		destDir := filepath.Join(outDir, outPath)
		createDir(destDir)
		copyFile(fileName, destDir)
		log.Info("Copied file: " + fileName)
	}
	if len(bucketName) > 0 {
		uploadFile(fileName, bucketName, outPath)
		log.Info("Uploaded file to bucket" + bucketName)
	}
	// TODO! Write index.html file
	return nil
}

// Loops through all files in a dir
func organiseFiles(inDirName, outDirName, bucketName string) {
	files, err := ioutil.ReadDir(inDirName)
	handleErr(err)

	for _, f := range files {
		fileName := inDirName + "/" + f.Name()

		// Get date taken for file
		date, err := getDateTaken(fileName)
		if err != nil {
			log.Warn(err.Error())
		}

		// Organise photo by moving to target folder
		err = processPhoto(fileName, outDirName, bucketName, date)
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func uploadFile(fileName, destName, bucketName string) error {
	// TODO! Upload file to a S3 bucket
	svc := s3.New(session.New(&aws.Config{Region: aws.String("ap-southeast-2")}))

	file, err := os.Open(fileName)

	if err != nil {
		log.Error(err)
	}

	defer file.Close()

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()

	buffer := make([]byte, size)

	// read file content to buffer
	file.Read(buffer)

	fileBytes := bytes.NewReader(buffer) // convert to io.ReadSeeker type

	fileType := http.DetectContentType(buffer)

	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName), // required
		Key:           aws.String(destName),   // required
		ACL:           aws.String("public-read"),
		Body:          fileBytes,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(fileType),
		Metadata: map[string]*string{
			"Key": aws.String("MetadataValue"), //required
		},
		// see more at http://godoc.org/github.com/aws/aws-sdk-go/service/s3#S3.PutObject
	}

	_, err = svc.PutObject(params)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// Generic AWS Error with Code, Message, and original error (if any)
			fmt.Println(awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
			if reqErr, ok := err.(awserr.RequestFailure); ok {
				// A service error occurred
				fmt.Println(reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
			}
		} else {
			// This case should never be hit, the SDK should always return an
			// error which satisfies the awserr.Error interface.
			fmt.Println(err.Error())
		}
	}
	return nil
}

func main() {

	// Declare a string parameter
	inDirNamePtr := flag.String("i", ".", "input directory")
	outDirNamePtr := flag.String("o", ".", "output directory")
	bucketNamePtr := flag.String("b", "", "bucket name")
	// Parse command line arguments.
	flag.Parse()
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}

	organiseFiles(*inDirNamePtr, *outDirNamePtr, *bucketNamePtr)
}
