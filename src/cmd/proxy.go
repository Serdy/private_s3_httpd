package cmd

import (
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
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

	// If the request path is exactly the prefix (e.g., /test/test2), list the S3 bucket contents
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
}
