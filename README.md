# Simple Server management for UNO project

It was build for usage on AWS Lambda.

## Build
```shell
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bin/bootstrap main.go;
```