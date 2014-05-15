@ECHO OFF
setlocal
SET GOPATH="%cd%"
cd src
go test
