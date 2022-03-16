set GOOS=linux
set GOARCH=amd64
statik -m -src="./web/dist" -f -dest="./server/embed" -p web -ns web
go build -ldflags "-s -w" -o Spark Spark/Server
@REM D:\TinyTools\UPX.exe -9 -v D:\WorkSpace\Web\Lab\Spark\Spark
