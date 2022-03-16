export GO111MODULE=auto



export GOOS=linux

export GOARCH=arm
go build -ldflags "-s -w" -o ./built/linux_arm Spark/Client
export GOARCH=arm64
go build -ldflags "-s -w" -o ./built/linux_arm64 Spark/Client
export GOARCH=386
go build -ldflags "-s -w" -o ./built/linux_i386 Spark/Client
export GOARCH=amd64
go build -ldflags "-s -w" -o ./built/linux_amd64 Spark/Client



export GOOS=windows

export GOARCH=arm
go build -ldflags "-s -w" -o ./built/windows_arm Spark/Client
export GOARCH=arm64
go build -ldflags "-s -w" -o ./built/windows_arm64 Spark/Client
export GOARCH=386
go build -ldflags "-s -w" -o ./built/windows_i386 Spark/Client
export GOARCH=amd64
go build -ldflags "-s -w" -o ./built/windows_amd64 Spark/Client



# export GOOS=android
# export CGO_ENABLED=1

# export GOARCH=arm
# export CC=armv7a-linux-androideabi21-clang
# export CXX=armv7a-linux-androideabi21-clang++
# go build -ldflags "-s -w" -o ./built/android_armv7a Spark/Client

# export GOARCH=arm64
# export CC=aarch64-linux-android21-clang
# export CXX=aarch64-linux-android21-clang++
# go build -ldflags "-s -w" -o ./built/android_aarch64 Spark/Client

# export GOARCH=386
# export CC=i686-linux-android21-clang
# export CXX=i686-linux-android21-clang++
# go build -ldflags "-s -w" -o ./built/android_i686 Spark/Client

# export GOARCH=amd64
# export CC=x86_64-linux-android21-clang
# export CXX=x86_64-linux-android21-clang++
# go build -ldflags "-s -w" -o ./built/android_x86_64 Spark/Client



statik -m -src="./built" -f -dest="./server/embed" -include=* -p built -ns built