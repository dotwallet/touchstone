protoc --go_out=plugins=grpc:message message/*.proto
go build