rm -Rf generated

mkdir -p generated

protoc --go_out=generated --go_opt=paths=source_relative \
    --go-grpc_out=generated --go-grpc_opt=paths=source_relative \
    --connect-go_out=generated --connect-go_opt=paths=source_relative, \
    *.proto
