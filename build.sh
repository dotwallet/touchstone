protoc --go_out=plugins=grpc:message message/*.proto
go get
go build