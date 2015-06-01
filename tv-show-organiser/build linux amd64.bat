@echo off
SET GOARCH=amd64
SET GOOS=linux
go build
SET GOOS=windows
SET GOARCH=386