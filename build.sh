#!/bin/bash
go build -o httpserver utils/httpserver/*.go || exit 1;
go build -o smppworker utils/smppworker/*.go || exit 1;
go build -o scheduler utils/scheduler/*.go || exit 1;
go build -o soapservice utils/soapservice/*.go || exit 1;