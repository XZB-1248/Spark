set GO111MODULE=auto
for /F %%i in ('git rev-parse HEAD') do ( set COMMIT=%%i)



set GOOS=darwin

set GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_darwin_arm64 Spark/server
set GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_darwin_amd64 Spark/server



set GOOS=linux

set GOARCH=arm
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_linux_arm Spark/Server
set GOARCH=386
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_linux_i386 Spark/Server
set GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_linux_arm64 Spark/Server
set GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_linux_amd64 Spark/Server



set GOOS=windows

set GOARCH=386
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_windows_i386.exe Spark/Server
set GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_windows_arm64.exe Spark/Server
set GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=%COMMIT%'" -tags=jsoniter -o ./releases/server_windows_amd64.exe Spark/Server
