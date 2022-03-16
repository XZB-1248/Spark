set GO111MODULE=auto



set GOOS=linux

set GOARCH=arm
go build -ldflags "-s -w" -o ./built/linux_arm Spark/Client
set GOARCH=arm64
go build -ldflags "-s -w" -o ./built/linux_arm64 Spark/Client
set GOARCH=386
go build -ldflags "-s -w" -o ./built/linux_i386 Spark/Client
set GOARCH=amd64
go build -ldflags "-s -w" -o ./built/linux_amd64 Spark/Client



set GOOS=windows

set GOARCH=arm
go build -ldflags "-s -w" -o ./built/windows_arm Spark/Client
set GOARCH=arm64
go build -ldflags "-s -w" -o ./built/windows_arm64 Spark/Client
set GOARCH=386
go build -ldflags "-s -w" -o ./built/windows_i386 Spark/Client
set GOARCH=amd64
go build -ldflags "-s -w" -o ./built/windows_amd64 Spark/Client



@REM set GOOS=android
@REM set CGO_ENABLED=1

@REM set GOARCH=arm
@REM set CC=armv7a-linux-androideabi21-clang
@REM set CXX=armv7a-linux-androideabi21-clang++
@REM go build -ldflags "-s -w" -o ./built/android_armv7a Spark/Client

@REM set GOARCH=arm64
@REM set CC=aarch64-linux-android21-clang
@REM set CXX=aarch64-linux-android21-clang++
@REM go build -ldflags "-s -w" -o ./built/android_aarch64 Spark/Client

@REM set GOARCH=386
@REM set CC=i686-linux-android21-clang
@REM set CXX=i686-linux-android21-clang++
@REM go build -ldflags "-s -w" -o ./built/android_i686 Spark/Client

@REM set GOARCH=amd64
@REM set CC=x86_64-linux-android21-clang
@REM set CXX=x86_64-linux-android21-clang++
@REM go build -ldflags "-s -w" -o ./built/android_x86_64 Spark/Client



statik -m -src="./built" -f -dest="./server/embed" -include=* -p built -ns built