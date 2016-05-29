# S3-photo-hosting
A command line utility that provides the following functionality:
 - Organise input photos and movies into a date ordered structure, great for transferring photos from camera to computer/backup.
 - Upload photos and movies to Amazon S3 for online backup.
 - Generates thumbnails and static web content to view photos online.
 - Checks if files exists before copying to save bandwidth (can be disabled using the -f command line)

# Static S3 website
A static website is generated and updated when photos are uploaded, allowing you to view your photos online or share them with family and friends. Photos/movies are ordered by date and have the following levels:
![Main Page](https://raw.githubusercontent.com/dylanclement/S3-photo-hosting/docs/docs/main.png)
Main page containing years.

![Yearly Page](https://raw.githubusercontent.com/dylanclement/S3-photo-hosting/docs/docs/yearly.png)
Yearly page with all dates and thumbnails.

![Main Page](https://raw.githubusercontent.com/dylanclement/S3-photo-hosting/docs/docs/daily.png)
Daily page with photos, clicking on one will open the full resolution image. 

It is fairly easy to set up DNS to host the static website on a custom domain, her is a guide, http://docs.aws.amazon.com/AmazonS3/latest/dev/website-hosting-custom-domain-walkthrough.html. ProTip! If you are planning on doing this, read through it as you do need to name your bucket correctly. If you already have a bucket and want to do this use the s3sync AWS cli utility to copy photos across buckets.

# Usage
The following command line flags are used.
 - -h - Prints command line usage.
 - -i (required) - Input directory for photos and movies.
 - -o (optional) - Output directory to copy files to in folders organised by date.
 - -b (optional) - Destination bucket name if uploading to S3.
 - -r (optional) - AWS region to use (defaults to us-east-1)
 - -f (optional) - Overwrite files if they already exist.

You will need to have an existing AWS account as well as provide credentials provide credentials (http://docs.aws.amazon.com/cli/latest/topic/config-vars.html) for the upload functionality to work.

# Compiling from source
## Prerequisites
 - Go 1.6+

Git clone into your GOPATH. Go to the folder containing main.go and install libraries using `go get`.
The command to build the command line app is `go build s3-utils.go templates.go photo-uploader.go`

# Disclaimer
This is a hobby project, feel free to contact me with any issues or better yet, submit a PR :) I can also not take responsibility for any problems that may arise from using this, I will not collect any personal information, the source code is there so have a look for yourself.

