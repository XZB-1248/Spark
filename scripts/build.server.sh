export GO111MODULE=auto
export COMMIT=`git rev-parse HEAD`



export GOOS=darwin

export GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_darwin_arm64 Spark/server
export GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_darwin_amd64 Spark/server



export GOOS=linux

export GOARCH=arm
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_linux_arm Spark/server
export GOARCH=386
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_linux_i386 Spark/server
export GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_linux_arm64 Spark/server
export GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_linux_amd64 Spark/server



export GOOS=windows

export GOARCH=386
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_windows_i386.exe Spark/server
export GOARCH=arm64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_windows_arm64.exe Spark/server
export GOARCH=amd64
go build -ldflags "-s -w -X 'Spark/server/config.COMMIT=$COMMIT'" -tags=jsoniter -o ./releases/server_windows_amd64.exe Spark/server
