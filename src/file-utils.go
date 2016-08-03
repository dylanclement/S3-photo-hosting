package main

import (
	//	"errors"
	"bytes"
	"image/jpeg"
	"io"
	"os"
	filepath "path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
)

// DefaultTime Returns a default time of 2000-01-01 UTC
func DefaultTime() time.Time {
	return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
}

// IsJpeg Checks whether a file is a jpeg
func IsJpeg(fileName string) bool {
	fileExt := strings.ToLower(filepath.Ext(fileName))
	return fileExt == ".jpg" // TODO! || fileExt == ".jpeg"
}

// IsMovie returns true is the file is a movie
func IsMovie(fileName string) bool {
	fileExt := strings.ToLower(filepath.Ext(fileName))
	return fileExt == ".mpg" || fileExt == ".mpeg" || fileExt == ".avi" || fileExt == ".mp4" || fileExt == ".3gp" || fileExt == ".mov"
}

// GetFileModTime Helper to get file modification time, useful as a fallback if file is not a jpg.
func GetFileModTime(fileName string) time.Time {
	var containsDateRegExp = regexp.MustCompile(`^(\d{8})_.*`)
	matches := containsDateRegExp.FindStringSubmatch(fileName)
	// if filename is eg. 20160513_181656.mp4 get the date from the filename instead
	if len(matches) > 0 {
		// useful if we re-encode a badly encoded camera movie, then we don't want to use the modified date
		dateStr := matches[1]
		date, _ := time.Parse("20060102", dateStr)
		return date
	}

	// else fetch the files last modification timne
	stat, err := os.Stat(fileName)
	if err != nil {
		log.Error("Unable to get ModTime for file: ", fileName)
		return DefaultTime()
	}
	return stat.ModTime()
}

// GetDateTaken Gets date taken of a file. If it is a jpg it will attempt to use EXIF data
func GetDateTaken(fileName string) time.Time {
	if len(fileName) <= 0 {
		return DefaultTime()
	}

	// Get the file extension for example .jpg
	if !IsJpeg(fileName) {
		// Get the current date and time for files that aren't photos
		return GetFileModTime(fileName)
	}

	// Make sure we can open the file
	file, err := os.Open(fileName)
	if err != nil {
		return DefaultTime()
	}
	defer file.Close()

	// Decode exif data from file
	var data *exif.Exif
	var date time.Time
	data, err = exif.Decode(file)
	if err != nil {
		// file might not have exif data, use os.Stat
		date = GetFileModTime(fileName)
	} else {
		// get date taken from exif data in jpg
		date, _ = data.DateTime()
	}

	return date
}

// CreateDir helper to create a folder if it doesn't exist
func CreateDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		// Ok directory doesn't exist, create it
		err := os.Mkdir(dirName, 0777)
		if err != nil {
			log.Error("Error creating directory: ", err.Error())
		}
	}
}

// CopyFile Helper function to copy a file
func CopyFile(src, dst string) error {
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

// CreateThumbNail Creates a thumbnail for an image
func CreateThumbNail(inFile string, width uint) ([]byte, error) {
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

// Gets the size of a file in bytes
func GetFileSize(fileName string) int64 {
	file, err := os.Open(fileName)
	if err != nil {
		log.Error(err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	return fileInfo.Size()
}

const thumbNailSize = 160
