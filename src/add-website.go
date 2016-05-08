package main

import (
	//	"errors"

	"encoding/json"
	"flag"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Every folder in the bucket has a json file that lists the files
func addFilesToBucket(svc s3.S3, sourceName, bucketName, fileName string) {

	res, _ := GetFromS3(svc, sourceName+"/photos.json", bucketName)

	// TODO! If it doesn't exist, create an empty buffer for res
	log.Info("Adding file ", fileName, " to source ", sourceName, " Res = ", res)

	// Read files from Json into map
	var dat map[string][]string
	if err := json.NewDecoder(res).Decode(&dat); err != nil {
		log.Fatal(err)
	}
	/*	buf := make([]byte, size)
		if _, err := io.ReadFull(res, buf); err != nil {
			log.Fatal(err)
		}
		if err := json.Unmarshal(buf, &dat); err != nil {
			panic(err)
		}*/

	// Add fileName
	var files = dat["files"]
	files = append(files, fileName)
	log.Info("Files = ", files, " obj = ")

	//newSlice := make([]string, len(files), 2*cap(files))
	//copy(newSlice, files)
	//newSlice[len(newSlice)] = fileName
	//dat["files"] = newSlice

	output, err := json.Marshal(dat)
	if err != nil {
		panic(err)
	}

	// Upload updated file to bucket
	UploadToS3(svc, sourceName+fileName, bucketName, output, int64(len(output)))
}

// Gets a list of all objects ina a S3 bucket
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
	log.Info("Results = ", json)
	return json
}

// Creates index.html to view photos
func createWebsite(svc s3.S3, bucketName, folderName string) error {
	template := `<!doctype html>
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
	  <!--a href="<%Back%>">Back</a-->
	  <div class="container" ng-controller="MainCtrl">
			<h1><%Title%></h1>
			</br>
	    <div class="navbar" />
	    <div class="body">
	      <div ng-repeat="filename in files">
	        <p>{{filename}}</p>
	        <a href="{{filename}}"><img ng-src="{{getThumbJpg(filename)}}" class="img-thumbnail" /></a>
	      </div>
	    </div>
	    <div class="footer">
	      <!--p class="muted">&copy; dylan clement 2016</p-->
	    </div>
	  </div>
	</body>
	</html>`
	test := strings.Replace(template, "<%Title%>", folderName, -1)
	UploadToS3(svc, folderName+"/index.html", bucketName, []byte(test), int64(len(test)))
	return nil
}

// processes all items in a bucket, creates an index and file.json
func processBucket(svc s3.S3, bucketName, folderName string) error {
	objects := getObjectsFromBucket(svc, bucketName, folderName)
	jsonFile := createJSONFile(svc, bucketName, folderName, objects)
	UploadToS3(svc, folderName+"/photos.json", bucketName, []byte(jsonFile), int64(len(jsonFile)))
	createWebsite(svc, bucketName, folderName)
	return nil
}

func main() {
	// Declare a string parameter

	bucketNamePtr := flag.String("b", "", "bucket name")
	awsRegionNamePtr := flag.String("r", "us-east-1", "AWS region")
	folderNamePtr := flag.String("f", "", "folder in bucket")
	// Parse command line arguments.
	flag.Parse()
	if len(*bucketNamePtr) == 0 || len(*bucketNamePtr) == 0 {
		log.Fatal("Error, need to pass in a bucket and folder name.")
	}
	svc := s3.New(session.New(&aws.Config{Region: aws.String(*awsRegionNamePtr)}))

	processBucket(*svc, *bucketNamePtr, *folderNamePtr)
	log.Info("Done processing: ", *bucketNamePtr)
}
