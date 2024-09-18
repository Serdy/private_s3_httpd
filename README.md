# private_s3_httpd Documentation

## Overview
`private_s3_httpd` is an HTTP server that proxies requests to an AWS S3 bucket. It allows listing objects in the bucket and serves individual objects as downloadable files. The server supports specifying a prefix for request paths and can log incoming HTTP requests.

## Features
- Serve and list objects from an S3 bucket via HTTP.
- Support for a configurable path prefix (`-prefix`).
- Optional logging of HTTP requests.
- Support for custom S3 endpoints (e.g., for MinIO).

---

## Prerequisites
- AWS credentials configured via environment variables or `~/.aws/credentials`.
- Go 1.22 or later installed.
- Access to the S3 bucket you want to proxy.

---

## Installation
Clone the repository and build the Go binary:

```bash
git clone https://github.com/serdy/private_s3_httpd.git
cd private_s3_httpd
go build -o private_s3_httpd main.go
```

## Usage

Run the server using the following flags:

```bash
./private_s3_httpd -bucket <S3_BUCKET_NAME> -listen <ADDRESS:PORT> -region <AWS_REGION> [-prefix <PREFIX>] [-log-requests] [-s3-endpoint <S3_ENDPOINT>]
```

### Flags:
- `-bucket`: The name of the S3 bucket to serve.
- `-listen`: The address and port on which the server will listen (default: `:8080`).
- `-region`: The AWS region where the S3 bucket is located (default: `us-east-1`).
- `-prefix`: An optional path prefix that maps incoming HTTP requests to a specific path in the S3 bucket.
- `-log-requests`: If set, logs all HTTP requests to the console.
- `-s3-endpoint`: An optional custom S3-compatible endpoint (e.g., for MinIO).

---

## Example

Run the server on `127.0.0.1:8888` to serve the bucket `another-bucket` in the `eu-central-1` region, with a request prefix of `test/test2`, and enable request logging:

```bash
./private_s3_httpd -bucket another-bucket -listen 127.0.0.1:8888 -region eu-central-1 -prefix test/test2 -log-requests
```

## cURL Example:

To list the contents of the bucket at the specified prefix:
```bash
curl http://127.0.0.1:8888/test/test2
```
This will list the objects in the S3 bucket under the path /test/test2.
