package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/rwcarlsen/goexif/exif"
	"io"
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
