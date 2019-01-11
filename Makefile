gen:
	@echo "generating protobufs..."
	@protoc \
    	--proto_path=${GOPATH}/src \
    	--proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
    	    --gogofaster_out=plugins=grpc,Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:./proto \
    	    --govalidators_out=gogoimport=true,Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:./proto \
    	--proto_path=./proto \
    	./proto/*.proto

build:
	go build ./cmd/centerms

run:
ifeq (,$(wildcard ./centerms))
	go build ./cmd/centerms
endif
	APP_ID=centerms TTL=4 RETRY=10 LOG_LEVEL=DEBUG STORE_HOST=127.0.0.1 STORE_PORT=6379 RPC_PORT=8090 REST_PORT=8080 WEBSOCKET_PORT=8070  ./centerms

clean:
	rm centerms

drun:
	docker run -p 50051:50051 centerms

swaggen:
	@cd ./cmd/centerms && \
	swagger generate spec -o ../../swagger.json

swagserv: 
	swagger serve -p=8090 -F=swagger swagger.json
