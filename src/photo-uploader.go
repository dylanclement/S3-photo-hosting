package main

import (
	//	"errors"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	filepath "path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
)

// Gets a list of all objects in a a S3 bucket
func getObjectsFromBucket(svc s3.S3, bucketName, folder string) []*s3.Object {
	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(folder),
	}

	resp, _ := svc.ListObjects(params)
	return resp.Contents
}

// Creates a file in the bucket to list the files
func createJSONFile(svc s3.S3, bucketName, folderName string, objects []*s3.Object) string {
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
func createWebsite(svc s3.S3, bucketName, folderName string, date time.Time) error {
	test := strings.Replace(websiteTemplate, "<%Title%>", folderName, -1)
	test = strings.Replace(test, "<%Back%>", date.Format("../../2006/index.html"), -1)
	UploadToS3(svc, folderName+"/index.html", bucketName, []byte(test), int64(len(test)))
	return nil
}

func updateFolderWebsite(svc s3.S3, bucketName string, date time.Time) error {
	// Create dates.json file
	dateYear := date.Format("2006")
	dateFull := date.Format("2006-01-02")
	datesFile := dateYear + "/dates.json"

	// Unmarshal into struct
	var dateStruct map[string][]string
	reader, _ := GetFromS3(svc, datesFile, bucketName)
	if reader == nil {
		// file doesn't exist, create it
		dateStruct = make(map[string][]string)
	} else {
		json.NewDecoder(reader).Decode(&dateStruct)
	}

	// Check if date exists in array
	found := false
	for _, dateF := range dateStruct["dates"] {
		if dateFull == dateF {
			found = true
		}
	}

	// Date doesn't exist in list
	if !found {
		dateStruct["dates"] = append(dateStruct["dates"], dateFull)
		sort.Strings(dateStruct["dates"])
		dateJSON, _ := json.Marshal(dateStruct)
		UploadToS3(svc, datesFile, bucketName, dateJSON, int64(len(dateJSON)))

		// Create index.html file
		test := strings.Replace(folderTemplate, "<%Title%>", dateYear, -1)
		UploadToS3(svc, dateYear+"/index.html", bucketName, []byte(test), int64(len(test)))
	}
	return nil
}

func updateMainWebsite(svc s3.S3, bucketName string) error {
	test := strings.Replace(mainTemplate, "<%Title%>", bucketName, -1)
	UploadToS3(svc, "index.html", bucketName, []byte(test), int64(len(test)))
	return nil
}

// processes all items in a bucket, creates an index and file.json
func processBucket(svc s3.S3, bucketName string, date time.Time) error {
	folderName := date.Format("2006/2006-01-02")
	objects := getObjectsFromBucket(svc, bucketName, folderName)
	jsonFile := createJSONFile(svc, bucketName, folderName, objects)
	UploadToS3(svc, folderName+"/photos.json", bucketName, []byte(jsonFile), int64(len(jsonFile)))
	createWebsite(svc, bucketName, folderName, date)
	updateFolderWebsite(svc, bucketName, date)
	updateMainWebsite(svc, bucketName)
	return nil
}

// Returns a default time of 2000-01-01 UTC
func defaultTime() time.Time {
	return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
}

// Checks whether a file is a jpeg
func isJpeg(fileName string) bool {
	fileExt := strings.ToLower(filepath.Ext(fileName))
	return fileExt == ".jpg" || fileExt == ".jpeg"
}

func isMovie(fileName string) bool {
	fileExt := strings.ToLower(filepath.Ext(fileName))
	return fileExt == ".mpg" || fileExt == ".mpeg" || fileExt == ".avi" || fileExt == ".mp4"
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
func getDateTaken(fileName string) time.Time {
	if len(fileName) <= 0 {
		return defaultTime()
	}

	// Get the file extension for example .jpg
	if !isJpeg(fileName) {
		// Get the current date and time for files that aren't photos
		return getFileModTime(fileName)
	}

	// Make sure we can open the file
	file, err := os.Open(fileName)
	if err != nil {
		return defaultTime()
	}
	defer file.Close()

	// Decode exif data from file
	var data *exif.Exif
	var date time.Time
	data, err = exif.Decode(file)
	if err != nil {
		// file might not have exif data, use os.Stat
		date = getFileModTime(fileName)
	} else {
		// get date taken from exif data in jpg
		date, _ = data.DateTime()
	}

	return date
}

// helper to create a folder if it doesn't exist
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

const thumbNailSize = 160

// Uploads a single file to S3. This needs to create a thumbnail, create update
//   the index.html for the folder and for the parent directory.
func uploadFile(svc s3.S3, sourceFile, outPath, fileName, bucketName string) error {
	file, err := os.Open(sourceFile)

	if err != nil {
		log.Error(err)
	}

	defer file.Close()

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()

	buffer := make([]byte, size)

	// read file content to buffer
	file.Read(buffer)

	destName := outPath + "/" + fileName // AWS uses forward slashes so don't use filePath.Join
	UploadToS3(svc, destName, bucketName, buffer, size)

	// If this is a photo create a thumbnail
	if isJpeg(sourceFile) {
		thumbBuf, thumbErr := createThumbNail(sourceFile, thumbNailSize)
		if thumbErr != nil {
			log.Error("Error creating thumbnail: ", err.Error())
		}
		// Upload
		UploadToS3(svc, outPath+"/"+filepath.Base(fileName)+"_thumb.jpg", bucketName, thumbBuf, int64(len(thumbBuf)))
	} else if isMovie(sourceFile) {
		cmd := exec.Command("ffmpeg", "-i", sourceFile, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", thumbNailSize, 1), "-f", "singlejpeg", "-")
		var buffer bytes.Buffer
		cmd.Stdout = &buffer
		if cmd.Run() != nil {
			log.Panic("Could not generate frame from movie ", sourceFile)
		}
		UploadToS3(svc, outPath+"/"+filepath.Base(fileName)+"_thumb.jpg", bucketName, buffer.Bytes(), int64(buffer.Len()))
	}

	return err
}

// Processes a single photo file, copying it to the output dir and creating thumbnails etc. in S3
func processFile(svc s3.S3, sourceFile, outDir, bucketName string, dateTaken time.Time) error {
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
		err := uploadFile(svc, sourceFile, outPath, fileName, bucketName)
		if err != nil {
			return err
		}

		log.Info("Uploaded file ", fileName, " to bucket: "+bucketName)
	}

	return nil
}

// Gets all files in directory
func addFilesToMap(inDirName string, fileMap map[time.Time][]string) {
	files, err := ioutil.ReadDir(inDirName)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, f := range files {
		if f.IsDir() {
			addFilesToMap(filepath.Join(inDirName, f.Name()), fileMap)
		} else {
			if isJpeg(f.Name()) || isMovie(f.Name()) {
				fileName := filepath.Join(inDirName, f.Name())
				dateTaken := getDateTaken(fileName)
				fileMap[dateTaken] = append(fileMap[dateTaken], fileName)
			}
		}
	}
}

// Loops through all files in a dir and processes them all
func process(svc *s3.S3, inDirName, outDirName, bucketName string) {
	// Get all files in directory
	fileMap := make(map[time.Time][]string)
	addFilesToMap(inDirName, fileMap)

	// Since we are using go routines to process the files, create channels and sync waits
	sem := make(chan int, 8) // Have 8 running concurrently
	var wg sync.WaitGroup

	for date, files := range fileMap {
		for _, fileName := range files {
			// Organise photo by moving to target folder or uploading it to S3

			wg.Add(1)
			go func(fileNameInner string, dateInner time.Time) {
				sem <- 1 // Wait for active queue to drain.
				err := processFile(*svc, fileNameInner, outDirName, bucketName, dateInner)
				if err != nil {
					log.Fatal(err.Error())
				}
				log.Info("Processed file: ", fileNameInner)

				wg.Done()
				<-sem // Done; enable next request to run.
			}(fileName, date)

		}
		wg.Wait() // Wait for all goroutines to finish
		processBucket(*svc, bucketName, date)
	}
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

	// Create S3 service
	svc := s3.New(session.New(&aws.Config{Region: aws.String(*awsRegionNamePtr)}))

	process(svc, *inDirNamePtr, *outDirNamePtr, *bucketNamePtr)
	log.Info("Done processing: ", *inDirNamePtr)
}

const websiteTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
	<title><%Title%></title>
	<link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
	<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
	<script type="text/javascript">
		var myApp = angular.module('myApp',[]);

		myApp.controller("MainCtrl", function($scope, $http, $q) {
			var res = $http.get("photos.json").then(function successCallback(results) {
				$scope.files = results.data.files;
			}, function errorCallback(response) {
				alert(response)
			})

			// gets thethumbnail name for the file
			$scope.getThumbJpg = function(fileName) {
				console.log("Test, " + fileName)
				var idx = fileName.lastIndexOf(".");
				return fileName.slice(0, idx) + "_thumb" + fileName.slice(idx);
			}
		});
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<a href="<%BACK%>"><%YEAR%></a><h2><%DATE%></h2>
		<div class="col-lg-12">

    </div>
		<div class="body">
			<div ng-repeat="filename in files">
				<div class="col-lg-3 col-md-4 col-xs-6 thumb">
					<a href="{{filename}}"><img ng-src="{{getThumbJpg(filename)}}" class="img-thumbnail" alt="{{filename}}"/></a>
				</div>
			</div>
		</div>
	</div>
</body>
</html>`

const folderTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
<title><%Title%></title>
<link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
<script type="text/javascript">
	var myApp = angular.module('myApp',[]);

	myApp.controller("MainCtrl", function($scope, $http, $q) {
		var res = $http.get("dates.json").then(function successCallback(results) {
			$scope.dates = results.data.dates;
		}, function errorCallback(response) {
			alert(response)
		})
	});
</script>
</head>
<body>
	<!--a href="<%Back%>">Back</a-->
	<div class="container" ng-controller="MainCtrl">
		<h1><%Title%></h1>
		</br>
		<div class="navbar" />
		<div class="body">
			<div ng-repeat="date in dates">
				<p>{{date}}</p>
				<a href="{{date}}/index.html"><img ng-src="http://findicons.com/files/icons/2221/folder/128/normal_folder.png" class="img-thumbnail" /></a>
			</div>
		</div>
	</div>
</div>
</body>
</html>`

const mainTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
  <title><%Title%></title>
  <link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
  <script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
  <script type="text/javascript">
    var myApp = angular.module('myApp',[]);

    myApp.controller("MainCtrl", function($scope, $http, $q) {
      var res = $http.get("years.json").then(function successCallback(results) {
        $scope.years = results.data.years;
      }, function errorCallback(response) {
        alert(response)
      })
    });
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<h1><%Title%></h1>
		</br>
		<div class="navbar" />
		<div class="body">
			<div ng-repeat="year in years">
				<p>{{year}}</p>
				<a href="{{year}}/index.html"><img ng-src="http://findicons.com/files/icons/2221/folder/128/normal_folder.png" class="img-thumbnail" /></a>
			</div>
		</div>
  </div>
</body>
</html>`
