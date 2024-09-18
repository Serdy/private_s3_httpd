package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/handlers"
	"github.com/serdy/private_s3_httpd/cmd"
)

func main() {
	showVersion := flag.Bool("version", false, "print version string")
	listen := flag.String("listen", ":8080", "address:port to listen on.")
	bucket := flag.String("bucket", "", "S3 bucket name")
	logRequests := flag.Bool("log-requests", true, "log HTTP requests")
	region := flag.String("region", "us-east-1", "AWS S3 Region")
	s3Endpoint := flag.String("s3-endpoint", "", "alternate http://address for accessing s3 (for configuring with minio.io)")
	prefix := flag.String("prefix", "", "Prefix for the HTTP paths")
	flag.Parse()

	if *showVersion {
		fmt.Printf("private_s3_httpd v%s (built w/%s)\n", cmd.VERSION, runtime.Version())
		return
	}

	if *bucket == "" {
		log.Fatalf("bucket name required")
	}

	var svc *s3.Client
	var cfg aws.Config
	var err error
	ctx := context.Background()

	if *s3Endpoint != "" {
		log.Printf("Using alternate S3 Endpoint %q with UsePathStyle:true", *s3Endpoint)

		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(*region),
		)
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		svc = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.EndpointResolver = s3.EndpointResolverFromURL(*s3Endpoint)
			o.UsePathStyle = true
		})

	} else {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(*region),
		)
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		svc = s3.NewFromConfig(cfg)
	}

	// Initialize Proxy with S3 client and bucket name
	proxy := &cmd.Proxy{
		Bucket: *bucket,
		Svc:    svc,
		Prefix: *prefix, // Pass the prefix here
	}

	// Declare h as an http.Handler and assign the Proxy to it
	var h http.Handler = proxy

	// If logRequests is enabled, wrap the handler with CombinedLoggingHandler
	if *logRequests {
		h = handlers.CombinedLoggingHandler(os.Stdout, h)
	}

	// Set up the HTTP server with the handler
	s := &http.Server{
		Addr:           *listen,
		Handler:        h,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Printf("listening on %s", *listen)
	log.Fatal(s.ListenAndServe())
}
