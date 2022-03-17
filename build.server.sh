mkdir ./releases



export GOOS=linux

export GOARCH=arm
go build -ldflags "-s -w" -o ./releases/server_linux_arm Spark/Server
export GOARCH=arm64
go build -ldflags "-s -w" -o ./releases/server_linux_arm64 Spark/Server
export GOARCH=386
go build -ldflags "-s -w" -o ./releases/server_linux_i386 Spark/Server
export GOARCH=amd64
go build -ldflags "-s -w" -o ./releases/server_linux_amd64 Spark/Server



export GOOS=windows

export GOARCH=arm
go build -ldflags "-s -w" -o ./releases/server_windows_arm Spark/Server
export GOARCH=arm64
go build -ldflags "-s -w" -o ./releases/server_windows_arm64 Spark/Server
export GOARCH=386
go build -ldflags "-s -w" -o ./releases/server_windows_i386 Spark/Server
export GOARCH=amd64
go build -ldflags "-s -w" -o ./releases/server_windows_amd64 Spark/Server