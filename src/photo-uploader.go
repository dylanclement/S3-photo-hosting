package main

import (
	//	"errors"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	filepath "path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var awsSession *session.Session
var overwrite = false
var keepMoviesOriginal = false

// TODO! Embed videos (http://stackoverflow.com/questions/10009918/how-can-i-embed-an-mpg-into-my-webpage)

// Creates a file in the bucket to list the files
func createJSONFile(bucketName, folderName string, objects []*s3.Object) string {
	var json = `{"files" : [`
	for idx, obj := range objects {
		fileName := strings.TrimPrefix(*obj.Key, folderName+"/")
		if fileName != "index.html" && fileName != "photos.json" && !strings.Contains(fileName, "_thumb.jpg") {
			if idx != 0 {
				json += ", "
			}
			json += `"` + fileName + `"`
		}
	}
	json += `]}`
	return json
}

// Creates index.html to view photos
func createWebsite(svc s3.S3, bucketName string, date time.Time) error {
	folderName := date.Format("2006/2006-01-02")
	test := strings.Replace(WebsiteTemplate, "<%Title%>", folderName, -1)
	test = strings.Replace(test, "<%BACK%>", date.Format("../../2006/index.html"), -1)
	test = strings.Replace(test, "<%YEAR%>", date.Format("2006"), -1)
	test = strings.Replace(test, "<%DATE%>", date.Format("2006-01-02"), -1)
	UploadToS3(svc, date.Format("2006/2006-01-02/index.html"), bucketName, []byte(test), int64(len(test)), true)
	return nil
}

// processes all items in a bucket, creates an index and file.json
func createJSONandWebsiteForFolder(svc s3.S3, bucketName string, folder time.Time) error {
	folderName := folder.Format("2006/2006-01-02")
	objects := GetObjectsFromBucket(svc, bucketName, folderName)
	jsonFile := createJSONFile(bucketName, folderName, objects)
	// Upload photos.json
	UploadToS3(svc, folderName+"/photos.json", bucketName, []byte(jsonFile), int64(len(jsonFile)), true)

	// Creates the index.html
	createWebsite(svc, bucketName, folder)

	// Creates the thumbnail from the first thumbnail
	thumbImg := "http://findicons.com/files/icons/2221/folder/128/normal_folder.png"
	for _, obj := range objects {
		fileName := strings.TrimPrefix(*obj.Key, folderName+"/")
		if strings.HasSuffix(fileName, "_thumb.jpg") {
			thumbImg = folder.Format("2006-01-02/") + fileName
			break
		}
	}

	// Add's the date to the folder website .json file, also passes in a thumbnail
	addDateToFolderWebsite(svc, bucketName, thumbImg, folder)

	// Finally update the main website
	addYearToMainWebsite(svc, bucketName, folder)
	return nil
}

type folderStruct struct {
	Date  string `json:"date"`
	Thumb string `json:"thumb"`
}

// NameSorter sorts planets by name.
type folderSorter []folderStruct

func (a folderSorter) Len() int           { return len(a) }
func (a folderSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a folderSorter) Less(i, j int) bool { return a[i].Date < a[j].Date }

func addDateToFolderWebsite(svc s3.S3, bucketName, thumb string, date time.Time) error {
	// Create dates.json file
	dateYear := date.Format("2006")
	dateFull := date.Format("2006-01-02")
	datesFile := dateYear + "/dates.json"

	// Unmarshal into struct
	var dateStruct map[string][]folderStruct
	reader := GetFromS3(svc, datesFile, bucketName)
	if reader == nil {
		// file doesn't exist, create it
		dateStruct = make(map[string][]folderStruct)
	} else {
		json.NewDecoder(reader).Decode(&dateStruct)
	}

	// Check if date exists in array
	found := false
	for _, dateF := range dateStruct["dates"] {
		if dateFull == dateF.Date {
			found = true
		}
	}

	// Date doesn't exist in list
	if !found {
		// Insert the first item
		s := folderStruct{dateFull, thumb}
		dateStruct["dates"] = append(dateStruct["dates"], s)
		sort.Sort(folderSorter(dateStruct["dates"]))
		dateJSON, _ := json.Marshal(dateStruct)
		UploadToS3(svc, datesFile, bucketName, dateJSON, int64(len(dateJSON)), true)

		// Create index.html file
		test := strings.Replace(FolderTemplate, "<%TITLE%>", dateYear, -1)
		UploadToS3(svc, dateYear+"/index.html", bucketName, []byte(test), int64(len(test)), overwrite)
	}
	return nil
}

func addYearToMainWebsite(svc s3.S3, bucketName string, date time.Time) error {
	// Create dates.json file
	dateYear := date.Format("2006")
	datesFile := "years.json"

	// Unmarshal into struct
	var dateStruct map[string][]string
	reader := GetFromS3(svc, datesFile, bucketName)
	if reader == nil {
		// file doesn't exist, create it
		dateStruct = make(map[string][]string)
	} else {
		json.NewDecoder(reader).Decode(&dateStruct)
	}

	// Check if date exists in array
	found := false
	for _, dateF := range dateStruct["years"] {
		if dateYear == dateF {
			found = true
		}
	}

	// Date doesn't exist in list
	if !found {
		// Insert the first item
		dateStruct["years"] = append(dateStruct["years"], dateYear)
		sort.Strings(dateStruct["years"])
		dateJSON, _ := json.Marshal(dateStruct)
		UploadToS3(svc, datesFile, bucketName, dateJSON, int64(len(dateJSON)), true)

		// Create index.html file
		test := strings.Replace(MainTemplate, "<%Title%>", bucketName, -1)
		UploadToS3(svc, "index.html", bucketName, []byte(test), int64(len(test)), overwrite)
	}
	return nil
}

// Uploads a single file to S3. This needs to create a thumbnail, create update
//   the index.html for the folder and for the parent directory.
func uploadFile(svc s3.S3, sourceFile, outPath, fileName, bucketName string) error {
	file, err := os.Open(sourceFile)

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()

	buffer := make([]byte, size)

	// read file content to buffer
	file.Read(buffer)

	destName := outPath + "/" + fileName // AWS uses forward slashes so don't use filePath.Join
	copied := UploadToS3(svc, destName, bucketName, buffer, size, overwrite)

	if !copied {
		// no need to upload thumbnail
		return nil
	}

	// If this is a photo create a thumbnail
	thumbFile := outPath + "/" + fileName[0:len(fileName)-4] + "_thumb.jpg"
	if IsJpeg(sourceFile) {
		thumbBuf, thumbErr := CreateThumbNail(sourceFile, thumbNailSize)
		if thumbErr != nil {
			log.Error("Error creating thumbnail: ", err.Error())
		}
		// Upload
		// TODO! Get length of extension, this won;t work for .JPEG files
		UploadToS3(svc, thumbFile, bucketName, thumbBuf, int64(len(thumbBuf)), overwrite)
	} else if IsMovie(sourceFile) {
		cmd := exec.Command("ffmpeg", "-i", sourceFile, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", thumbNailSize, thumbNailSize/4*3), "-f", "singlejpeg", "-")
		var buffer bytes.Buffer
		cmd.Stdout = &buffer
		if cmd.Run() != nil {
			log.Panic("Could not generate frame from movie ", sourceFile)
		}
		UploadToS3(svc, thumbFile, bucketName, buffer.Bytes(), int64(buffer.Len()), overwrite)
	}

	return err
}

// shrink a movie file
func shrinkMovie(sourceFile, tmpDir string, dateTaken time.Time) string {
	// Get an output file name, make all files mp4  and make sure we can support multiple files in the same dir
	destFile := filepath.Join(tmpDir, dateTaken.Format("20060102_150405")+".mp4")
	for i := 1; ; i++ {
		if _, err := os.Stat(destFile); os.IsNotExist(err) {
			break
		}
		destFile = filepath.Join(tmpDir, fmt.Sprintf(dateTaken.Format("20060102_150405")+"_%04d.mp4", i))
	}

	// Run ffmpeg on the input file and save to output dir
	cmd := exec.Command("ffmpeg", "-i", sourceFile, "-c:v", "libx264", "-preset", "medium", "-crf", "28", "-movflags", "+faststart", "-acodec", "aac", "-strict", "experimental", "-ab", "96k", destFile)
	if err := cmd.Run(); err != nil {
		log.Error("Could not run ffmpeg on file: ", sourceFile, err, destFile)
	}
	if err := os.Chtimes(destFile, dateTaken, dateTaken); err != nil {
		log.Error(err)
	}

	// Check what the ratio input/output is
	inSize := GetFileSize(sourceFile)
	outSize := GetFileSize(destFile)
	ratio := float64(outSize) / float64(inSize)
	if ratio < 0.93 {
		// new file is smaller, use that as the new destination
		log.Info("Using shrunk movie file (", ratio, "): ", destFile, " instead of ", sourceFile)
		return destFile
	}
	log.Info("Ratio not good (", ratio, "), using: ", sourceFile)
	return sourceFile
}

// Processes a single photo file, copying it to the output dir and creating thumbnails etc. in S3
func processFile(svc s3.S3, sourceFile, outDir, tmpDir, bucketName string, dateTaken time.Time) error {
	outPath := dateTaken.Format("2006/2006-01-02")
	fileName := strings.Replace(filepath.Base(sourceFile), " ", "", -1)

	if IsMovie(sourceFile) && !keepMoviesOriginal {
		// Shrink movie
		defer func() {
			if r := recover(); r != nil {
				log.Info("Recovered from error, retrying", r)
				sourceFile = shrinkMovie(sourceFile, tmpDir, dateTaken)
			}
		}()
		sourceFile = shrinkMovie(sourceFile, tmpDir, dateTaken)
	}
	// Sleep to avoid a race condition
	time.Sleep(100 * time.Millisecond)

	// If we specified a output folder, organise files
	if len(outDir) > 0 {
		CreateDir(filepath.Join(outDir, dateTaken.Format("2006")))
		CreateDir(filepath.Join(outDir, dateTaken.Format("2006/2006-01-02"))) // Can't created nested directories in one go
		destPath := filepath.Join(outDir, outPath, fileName)

		err := CopyFile(sourceFile, destPath)
		if err != nil {
			return err
		}

		log.Info("Copied file: ", destPath)
	}

	// If we passed in a bucket, upload to S3
	if len(bucketName) > 0 {
		err := uploadFile(svc, sourceFile, outPath, fileName, bucketName)
		if err != nil {
			return err
		}
	}

	return nil
}

// Gets all files in directory
func addFilesToMap(inDirName string, fileMap map[string][]string) {
	files, err := ioutil.ReadDir(inDirName)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, f := range files {
		if f.IsDir() {
			dirName := f.Name()
			if dirName[0] == '.' {
				continue
			}
			addFilesToMap(filepath.Join(inDirName, dirName), fileMap)
		} else {
			if IsJpeg(f.Name()) || IsMovie(f.Name()) {
				fileName := filepath.Join(inDirName, f.Name())
				dateTaken := GetDateTaken(fileName)
				dateKey := dateTaken.Format("2006-01-02")
				fileMap[dateKey] = append(fileMap[dateKey], fileName)
			}
		}
	}
}

// Remove any files from map already existing in S3
func removeExisting(svc s3.S3, bucketName string, fileMap map[string][]string) {
	for dateKey, files := range fileMap {
		date, _ := time.Parse("2006-01-02", dateKey)
		folderName := date.Format("2006/2006-01-02")
		s3Objs := GetObjectsFromBucket(svc, bucketName, folderName)
		var newFiles []string

		// Loop through files and S3 objects, if the file exists add it to a new array
		for _, fileName := range files {
			found := false
			for _, obj := range s3Objs {
				s3FileName := strings.TrimPrefix(*obj.Key, folderName+"/")
				if filepath.Base(fileName) == s3FileName {
					found = true
					log.Info("File ", fileName, " already exists on S3, skipping...")
					break
				}
			}
			if !found {
				newFiles = append(newFiles, fileName)
			}
		}
		// Replace old list with new one
		if len(newFiles) <= 0 {
			log.Info("Nothing to add for ", dateKey, " so removing from list.")
			delete(fileMap, dateKey) // remove key if all files are already on
		} else {
			log.Info("Adding newFiles ", newFiles)
			fileMap[dateKey] = newFiles
		}
	}
}

// Loops through all files in a dir and processes them all
func process(svc *s3.S3, inDirName, outDirName, bucketName string) {
	// Get all files in directory
	fileMap := make(map[string][]string)
	addFilesToMap(inDirName, fileMap)
	if !overwrite {
		removeExisting(*svc, bucketName, fileMap)
	}

	// Create temp dir and remember to clean up
	tmpDir, _ := ioutil.TempDir("", "shrink-file")
	defer os.RemoveAll(tmpDir) // clean up

	numDirs := len(fileMap)
	var doneDirs = 0
	for dateKey, files := range fileMap {

		// Get date from dateKey (ignoring time taken for photo)
		date, err := time.Parse("2006-01-02", dateKey)
		if err != nil {
			log.Error("Error parsing date: ", dateKey)
			continue
		}

		// Since we are using go routines to process the files, create channels and sync waits
		/*
			sem := make(chan int, 8) // Have 8 running concurrently
			var wg sync.WaitGroup

			// Organise photos by moving to target folder or uploading it to S3
			for _, fileName := range files {

				// Remember to increment the waitgroup by 1 BEFORE starting the goroutine
				wg.Add(1)
				go func(fileNameInner string, dateInner time.Time) {
					sem <- 1 // Wait for active queue to drain.
					err := processFile(*svc, fileNameInner, outDirName, tmpDir, bucketName, dateInner)
					if err != nil {
						log.Fatal(err.Error())
					}

					wg.Done()
					<-sem // Done; enable next request to run.
				}(fileName, date)

			}
			wg.Wait() // Wait for all goroutines to finish
		*/

		// Can't get goroutines working, not much of a speed improvement as the main bottleneck is AWS uploads.
		for _, fileName := range files {
			err := processFile(*svc, fileName, outDirName, tmpDir, bucketName, date)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		doneDirs++
		log.Info("Processed ", doneDirs, " of ", numDirs, " folders.")
		createJSONandWebsiteForFolder(*svc, bucketName, date)
	}
}

func main() {
	// Set up log file
	logFile, err := os.OpenFile("photo-uploader.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Error(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Set variables
	inDirNamePtr := flag.String("i", "", "input directory")
	outDirNamePtr := flag.String("o", "", "output directory")
	bucketNamePtr := flag.String("n", "", "bucket name")
	awsRegionNamePtr := flag.String("r", "us-east-1", "AWS region")
	flag.BoolVar(&overwrite, "f", false, "overwrite")
	flag.BoolVar(&keepMoviesOriginal, "k", true, "don't shrink movies")
	// Parse command line arguments.
	flag.Parse()
	log.Info("Overwrite: ", overwrite)
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}

	// Create S3 service
	awsSession = session.New(&aws.Config{Region: aws.String(*awsRegionNamePtr)})
	svc := s3.New(awsSession)

	process(svc, *inDirNamePtr, *outDirNamePtr, *bucketNamePtr)
	log.Info("Done processing: ", *inDirNamePtr)
}
