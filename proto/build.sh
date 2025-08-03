rm -Rf schema

mkdir -p schema

protoc --go_out=schema --go_opt=paths=source_relative \
    --go-grpc_out=schema --go-grpc_opt=paths=source_relative \
    *.proto
