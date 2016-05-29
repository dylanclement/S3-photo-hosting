# S3-photo-hosting
A command line utility that provides the following functionality:
 - Organise input photos and movies into a date ordered structure, great for transferring photos from camera to computer/backup.
 - Upload photos and movies to Amazon S3 for online backup.
 - Generates thumbnails and static web content to view photos online, can send the link to family and friends.
 - Checks if files exiost before copying to save bandwidth (can be disabled using the -f command line)

# Usage
The following command line flags are used.
 - -h - Prints command line usage.
 - -i (required) - Input directory for photos and movies.
 - -o (optional) - Output directory to copy files to in folders organised by date.
 - -b (optional) - Destination bucket name if uploading to S3.
 - -r (optional) - AWS region to use (defaults to us-east-1)
 - -f (optional) - Overwrite files if they already exist.

# Compiling from source
## Prerequisites
 - Go 1.6+

Git clone into your GOPATH. Go to the folder containing main.go and install libraries using `go get`.
The command to build the command line app is `go build s3-utils.go templates.go photo-uploader.go`
