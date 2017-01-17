#!/bin/bash
version=$(git show -s --pretty='format:%H')
GOOS=linux go build -o httpserver -ldflags="-X main.version=${version}" utils/httpserver/*.go || exit 1;
GOOS=linux go build -o smppworker -ldflags="-X main.version=${version}" utils/smppworker/*.go || exit 1;
GOOS=linux go build -o scheduler -ldflags="-X main.version=${version}" utils/scheduler/*.go || exit 1;
GOOS=linux go build -o soapservice -ldflags="-X main.version=${version}" utils/soapservice/*.go || exit 1;