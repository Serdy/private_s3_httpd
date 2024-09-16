package main

import (
	// "context"
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
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	rawKey := req.URL.Path

	// Remove the leading slash from the key
	key := strings.TrimPrefix(rawKey, "/")

	if key == "" {
		// List items in the bucket
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
			escapedKey := html.EscapeString(*obj.Key)
			fmt.Fprintf(rw, "<li><a href=\"%s\">%s</a></li>", escapedKey, escapedKey)
		}
		fmt.Fprintf(rw, "</ul></body></html>")
		return
	}

	if strings.HasSuffix(key, "/") {
		key = key + "index.html"
	}

	input := &s3.GetObjectInput{
		Bucket: &p.Bucket,
		Key:    &key,
	}
	if v := req.Header.Get("If-None-Match"); v != "" {
		input.IfNoneMatch = &v
	}

	resp, err := p.Svc.GetObject(ctx, input)
	var is304 bool
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NotModified":
				is304 = true
			case "NoSuchKey":
				http.Error(rw, "Page Not Found", http.StatusNotFound)
				return
			default:
				log.Printf("Error getting object %s: %v", key, apiErr)
				http.Error(rw, "Internal Error", http.StatusInternalServerError)
				return
			}
		} else {
			// Handle non-API errors
			log.Printf("Unknown error getting object %s: %v", key, err)
			http.Error(rw, "Internal Error", http.StatusInternalServerError)
			return
		}
	}

	var contentType string
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	if contentType == "" {
		ext := path.Ext(key)
		contentType = mime.TypeByExtension(ext)
	}

	if resp.ETag != nil && *resp.ETag != "" {
		rw.Header().Set("Etag", *resp.ETag)
	}

	if contentType != "" {
		rw.Header().Set("Content-Type", contentType)
	}

	if resp.ContentLength != nil {
		rw.Header().Set("Content-Length", fmt.Sprintf("%d", *resp.ContentLength))
	}

	// Set the Content-Disposition header to attachment
	filename := path.Base(key)
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if is304 {
		rw.WriteHeader(http.StatusNotModified)
	} else {
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Error copying response body: %v", err)
		}
	}
}
