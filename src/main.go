package main

import (
	"errors"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/rwcarlsen/goexif/exif"
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

// Helper function to copy a file
func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil { return err }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil { return err }
    defer out.Close()
    _, err = io.Copy(out, in)
    cerr := out.Close()
    if err != nil { return err }
    return cerr
}

func organisePhoto(currDirName, currFileName, outDir string, dateTaken time.Time) error {
	src := currDirName + currFileName
	dstDir := outDir + "\\" + dateTaken.Format("2006-01-02")
	dst := dstDir + "\\" + currFileName

	// Create the output directory
	createDir(dstDir)

	// Copy the file to the dest dir
	copyFile(src, dst)

	log.Info(src, " ==> ", dst)
	return nil
}


func uploadS3(fileName string, dateTaken time.Time) error {
	// TODO! Upload file to a S3 bucket
	return nil
}

func main() {

	// Declare a string parameter
	inDirNamePtr := flag.String("in", ".", "input directory")
	outDirNamePtr := flag.String("out", "", "output directory")
	// Parse command line arguments.
	flag.Parse()
	if len(*inDirNamePtr) == 0 {
		log.Fatal("Error, need to define an input directory.")
	}
	if len(*outDirNamePtr) == 0 {
		log.Fatal("Error, need to define an output directory.")
	}

	getFilesInDir(*inDirNamePtr, *outDirNamePtr)

}
