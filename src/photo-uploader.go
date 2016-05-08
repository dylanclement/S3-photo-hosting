package main

import (
	//	"errors"
	"bytes"
	"errors"
	"flag"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	filepath "path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
)

// Returns a default time of 2000-01-01 UTC
func defaultTime() time.Time {
	return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
}

// Helper to get file modification time, useful as a fallback if file is not a jpg.
func getFileModTime(fileName string) time.Time {
	stat, err := os.Stat(fileName)
	if err != nil {
		log.Error("Unable to get ModTime for file: ", fileName)
		return defaultTime()
	}
	return stat.ModTime()
}

// Get date taken of a file. If it is a jpg it will attempt to use EXIF data
func getDateTaken(fileName string) (time.Time, error) {
	if len(fileName) <= 0 {
		return defaultTime(), errors.New("Invalid filename passed.")
	}

	file, err := os.Open(fileName)
	if err != nil {
		return defaultTime(), err
	}

	// Get the file extension for example .jpg
	fileExt := strings.ToLower(filepath.Ext(fileName))

	if fileExt != ".jpg" {
		// Get the current date and time for files that aren't photos
		return getFileModTime(fileName), nil
	}

	var date time.Time
	var data *exif.Exif

	data, err = exif.Decode(file)
	if err != nil {
		// file might not have exif data, use os.Stat
		date = getFileModTime(fileName)
	} else {
		date, _ = data.DateTime()
	}

	return date, nil
}

// Create a folder
func createDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// Ok directory doesn't exist, create it
		err := os.Mkdir(dirName, 0777)
		if err != nil {
			log.Error("Error creating directory: ", err.Error())
		}
	}
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	// open input file
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// create dest file
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// copy contents from source to destination
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

// Processes a single photo file, copying it to the output dir and creating thumbnails etc. in S3
func processPhoto(svc s3.S3, sourceFile, outDir, bucketName, awsRegion string, dateTaken time.Time) error {
	outPath := dateTaken.Format("2006/2006-01-02")
	fileName := filepath.Base(sourceFile)

	// If we specified a output folder, organise files
	if len(outDir) > 0 {
		createDir(filepath.Join(outDir, dateTaken.Format("2006")))
		createDir(filepath.Join(outDir, dateTaken.Format("2006/2006-01-02"))) // Can't created nested directories in one go
		destPath := filepath.Join(outDir, outPath, fileName)

		err := copyFile(sourceFile, destPath)
		if err != nil {
			return err
		}

		log.Info("Copied file: ", destPath)
	}

	// If we passed in a bucket, upload to S3
	if len(bucketName) > 0 {
		destPath := outPath + "/" + fileName // AWS uses forward slashes so don't use filePath.Join
		err := uploadFile(svc, sourceFile, destPath, bucketName)
		if err != nil {
			return err
		}

		log.Info("Uploaded file to bucket: " + bucketName)
	}

	return nil
}

// Creates a thumbnail for an image
func createThumbNail(inFile string, width uint) ([]byte, error) {
	file, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	// decode jpeg into image.Image
	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, err
	}

	// resize to width using Lanczos resampling and preserve aspect ratio
	m := resize.Resize(width, 0, img, resize.Lanczos3)

	out := new(bytes.Buffer)

	// write new image to buffer
	jpeg.Encode(out, m, nil)

	log.Info("Created thumbnail for file: ", inFile)
	return out.Bytes(), nil
}

// Uploads a single file to S3. This needs to create a thumbnail, create update
//   the index.html for the folder and for the parent directory.
func uploadFile(svc s3.S3, fileName, destName, bucketName string) error {
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

	UploadToS3(svc, destName, bucketName, buffer, size)

	// If this is a photo create a thumbnail
	if strings.ToLower(filepath.Ext(fileName)) == ".jpg" {
		thumbBuf, thumbErr := createThumbNail(fileName, 128)
		if thumbErr != nil {
			log.Error("Error creating thumbnail: ", err.Error())
		}
		// Upload
		UploadToS3(svc, strings.Replace(destName, ".jpg", "_thumb.jpg", 1), bucketName, thumbBuf, int64(len(thumbBuf)))
	}
	// TODO! create a thumbnail for movies
	// TODO! upload index.html

	return err
}

// Gets all files in directory
func getFiles(inDirName string) []string {
	files, err := ioutil.ReadDir(inDirName)
	if err != nil {
		log.Fatal(err.Error())
	}

	numFiles := len(files)
	fileArr := make([]string, numFiles)
	for idx, f := range files {
		fileArr[idx] = inDirName + "/" + f.Name()
	}

	return fileArr
}

// Loops through all files in a dir and processes them all
func process(inDirName, outDirName, bucketName, awsRegion string) {
	// Get all files in directory
	files := getFiles(inDirName)
	// Create S3 service
	svc := s3.New(session.New(&aws.Config{Region: aws.String(awsRegion)}))

	for _, fileName := range files {
		// Get date taken for file
		date, err := getDateTaken(fileName)
		if err != nil {
			log.Warn(err.Error())
		} else {
			// Organise photo by moving to target folder or uploading it to S3
			err = processPhoto(*svc, fileName, outDirName, bucketName, awsRegion, date)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		log.Info("Processed file: ", fileName)
	}
	log.Info("Done processing: ", inDirName)
}

func main() {
	// Declare a string parameter

	inDirNamePtr := flag.String("i", "", "input directory")
	outDirNamePtr := flag.String("o", "", "output directory")
	bucketNamePtr := flag.String("b", "", "bucket name")
	awsRegionNamePtr := flag.String("r", "us-east-1", "AWS region")
	// Parse command line arguments.
	flag.Parse()
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}

	process(*inDirNamePtr, *outDirNamePtr, *bucketNamePtr, *awsRegionNamePtr)
	log.Info("Done processing: ", *inDirNamePtr)

}
