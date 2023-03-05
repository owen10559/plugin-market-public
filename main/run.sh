go env -w GOPROXY=https://goproxy.cn,direct
go mod init main
go mod tidy
go build main
./main
