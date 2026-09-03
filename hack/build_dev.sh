#!/bin/sh
go build -gcflags="all=-N -l" -o ./tmp/main .
