#!/bin/bash
GOOS=linux go build -o httpserver utils/httpserver/*.go || exit 1;
GOOS=linux go build -o smppworker utils/smppworker/*.go || exit 1;
GOOS=linux go build -o scheduler utils/scheduler/*.go || exit 1;
GOOS=linux go build -o soapservice utils/soapservice/*.go || exit 1;