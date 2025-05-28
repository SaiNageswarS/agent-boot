Remove-Item generated -Recurse
New-Item -Path . -Name "generated" -ItemType "directory"

cd ..\proto
protoc --go_out=..\search-core\generated --go_opt=paths=source_relative `
    --go-grpc_out=..\search-core\generated --go-grpc_opt=paths=source_relative `
    *.proto
cd ..\search-core

go build -mod=mod -o build/ .
