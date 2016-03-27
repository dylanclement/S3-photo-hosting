package main

import (
	"github.com/rwcarlsen/goexif/exif"
//    "github.com/rwcarlsen/goexif/mknote"
//    "github.com/xiam/exif"

    "errors"
    "fmt"
    "log"
	"os"
    "time"
)

func handleErr(err error) {
	if err != nil {
        fmt.Printf("Error:%s\n", err.Error())
		os.Exit(1)
	}
}

func getDateTaken(fileName string) (time.Time, error) {

	if len(fileName) <= 0 {
		log.Print("Pass filename as parameter.")
		return time.Now(), errors.New("Invalid filename passed.")
	}

	file, err := os.Open(fileName)
	if err != nil { return time.Now(), err }

	log.Printf("filename = %s\n", fileName)

    date := time.Now()
	data, err := exif.Decode(file)
    if err != nil {
        // file might noit have exif data, use os.Stat
        stat, err := os.Stat(fileName)
        handleErr(err)

        date = stat.ModTime()
    } else {
        date, _ = data.DateTime()
    }

    return date, err
} 
func main() {
	args := os.Args[1:]

    if len(args) < 1 {
        fmt.Printf("Please pass a filename as a parameter")
        os.Exit(1)
    }

    fileName := args[0]

    date, _ := getDateTaken(fileName)

    fmt.Printf("Date created = %s", date)
}
