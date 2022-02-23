#!/bin/sh

go test ./... -covermode=count -coverprofile=cover.out fmt
go tool cover -func=cover.out -o=cover.out