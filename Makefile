# go get github.com/gogo/protobuf/{proto,protoc-gen-gogo,gogoproto,protoc-gen-gofast,protoc-gen-gogofaster}
proto_gen:
	@echo "generating protobufs..."
	@protoc \
        --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        ./proto/*.proto

# go get github.com/golang/mock/gomock
# go get github.com/golang/mock/mockgen
mock_gen:
	@echo "generating mocks..."
	@mockgen github.com/kostiamol/centerms/api CfgProvider,DataProvider > ./api/mock/api.go
	@mockgen github.com/kostiamol/centerms/log Logger > ./log/mock/logger.go
	@mockgen github.com/kostiamol/centerms/svc CfgStorer,DataStorer,Publisher > ./svc/mock/svc.go

build:
	@go build -o centerms

test:
	@echo "running tests..."
	@go test ./...

lint:
	@echo "running linters..."
	@golangci-lint run
run:
	@./centerms

clean:
	@rm centerms

docker_lint:
	@hadolint Dockerfile

docker_build:
	@docker build -t kostiamol/centerms:latest .

docker_push:
	@docker push kostiamol/centerms:latest

docker_up:
	@docker-compose up

docker_down:
	@docker-compose down
