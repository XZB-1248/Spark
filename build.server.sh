statik -m -src="./web/dist" -f -dest="./server/embed" -p web -ns web
mkdir ./releases



export GOOS=linux

export GOARCH=arm
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_linux_arm Spark/server
export GOARCH=arm64
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_linux_arm64 Spark/server
export GOARCH=386
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_linux_i386 Spark/server
export GOARCH=amd64
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_linux_amd64 Spark/server



export GOOS=windows

export GOARCH=arm
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_windows_arm.exe Spark/server
export GOARCH=arm64
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_windows_arm64.exe Spark/server
export GOARCH=386
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_windows_i386.exe Spark/server
export GOARCH=amd64
go build -ldflags "-s -w" -tags=jsoniter -o ./releases/server_windows_amd64.exe Spark/server