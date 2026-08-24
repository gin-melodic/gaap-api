@echo off
go build -gcflags="all=-N -l" -o ./tmp/main.exe .
