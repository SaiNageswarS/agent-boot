rm -Rf generated

mkdir -p generated/mcp/v1
cd proto/mcp/v1
protoc --go_out=../../../generated/mcp/v1 --go_opt=paths=source_relative \
    --go-grpc_out=../../../generated/mcp/v1 --go-grpc_opt=paths=source_relative \
    *.proto
cd ../../.. 

mkdir -p generated/pb
cd proto 
protoc --go_out=../generated/pb --go_opt=paths=source_relative \
    --go-grpc_out=../generated/pb --go-grpc_opt=paths=source_relative \
    *.proto
cd ..

go build -mod=mod -o build/ .