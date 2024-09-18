package cmd

import (
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type Proxy struct {
	Bucket string
	Svc    *s3.Client
	Prefix string // Add Prefix field to the Proxy struct
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	rawKey := req.URL.Path

	// Check if the URL starts with the specified prefix
	if !strings.HasPrefix(rawKey, "/"+p.Prefix) {
		http.Error(rw, "Not Found", http.StatusNotFound)
		return
	}

	// Trim the prefix from the key
	key := strings.TrimPrefix(rawKey, "/"+p.Prefix)
	key = strings.TrimPrefix(key, "/") // Remove any leading slash

	// If the request path is exactly the prefix (e.g., /test/test2), list the S3 bucket contents
	if key == "" {
		// List all items in the bucket (do not set Prefix)
		input := &s3.ListObjectsV2Input{
			Bucket: &p.Bucket,
		}

		resp, err := p.Svc.ListObjectsV2(ctx, input)
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) {
				log.Printf("Error listing objects: %v", apiErr)
				http.Error(rw, "Internal Error", http.StatusInternalServerError)
				return
			} else {
				log.Printf("Unknown error listing objects: %v", err)
				http.Error(rw, "Internal Error", http.StatusInternalServerError)
				return
			}
		}

		// Create a simple HTML page listing the objects
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(rw, "<html><body><h1>Contents of Bucket %s</h1><ul>", html.EscapeString(p.Bucket))
		for _, obj := range resp.Contents {
			// For each object, create a link to its key
			objectKey := *obj.Key
			escapedKey := html.EscapeString(objectKey)
			urlPath := fmt.Sprintf("/%s/%s", p.Prefix, escapedKey)
			fmt.Fprintf(rw, "<li><a href=\"%s\">%s</a></li>", html.EscapeString(urlPath), escapedKey)
		}
		fmt.Fprintf(rw, "</ul></body></html>")
		return
	}

	// Handle downloading of individual files
	// Build the full key including the key
	fullKey := key

	// Get the object from S3
	input := &s3.GetObjectInput{
		Bucket: &p.Bucket,
		Key:    &fullKey,
	}

	resp, err := p.Svc.GetObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NoSuchKey":
				http.Error(rw, "File Not Found", http.StatusNotFound)
				return
			default:
				log.Printf("Error getting object %s: %v", fullKey, apiErr)
				http.Error(rw, "Internal Error", http.StatusInternalServerError)
				return
			}
		} else {
			// Handle non-API errors
			log.Printf("Unknown error getting object %s: %v", fullKey, err)
			http.Error(rw, "Internal Error", http.StatusInternalServerError)
			return
		}
	}
	defer resp.Body.Close()

	// Set appropriate headers
	var contentType string
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	} else {
		// Guess the content type based on the file extension
		ext := path.Ext(fullKey)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	rw.Header().Set("Content-Type", contentType)
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(fullKey)))

	// Copy the response body to the client
	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}
