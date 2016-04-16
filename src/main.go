package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/rwcarlsen/goexif/exif"
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
	log.Info("filename = ", fileName, " filePath = ", fileExt)

	date := time.Now()

	if fileExt == ".jpg" {

		data, err := exif.Decode(file)
		if err != nil {
			// file might not have exif data, use os.Stat
			date = getFileModTime(fileName)
		} else {
			log.Debug("Got here")
			date, _ = data.DateTime()
		}
	} else {
		date = getFileModTime(fileName)
	}

	return date, err
}

func getFilesInDir(dirName string) {
	files, err := ioutil.ReadDir(dirName)
	handleErr(err)

	for _, f := range files {
		fileName := dirName + "/" + f.Name()
		date, err := getDateTaken(fileName)
		if err != nil {
			log.Error("Error occured opening ", fileName, err.Error())
		}

		log.Info("Date created = ", date)
	}
}

func main() {
	log.SetOutput(os.Stdout)
	args := os.Args[1:]

	if len(args) < 1 {
		log.Fatal("Please folder name as a parameter")
	}

	dirName := args[0]

	getFilesInDir(dirName)

}
