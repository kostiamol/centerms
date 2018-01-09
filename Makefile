build:
	protoc  -I api/pb/ api/pb/api.proto \
		-I/usr/local/include \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:api/pb \
		--swagger_out=logtostderr=true:api/pb

