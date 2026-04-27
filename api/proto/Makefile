.PHONY: help generate generate-openapi clean deps

GRPC_GATEWAY_VERSION := v2.28.0
# googleapis protos vendored локально (grpc-gateway v2 не включает third_party)
GOOGLEAPIS_PATH := proto/third_party/googleapis

# Default target
help:
	@echo "Available targets:"
	@echo "  generate         - Generate Go code from proto files (run from api/proto)"
	@echo "  generate-openapi - Generate OpenAPI (swagger.json) from proto files"
	@echo "  clean            - Remove generated Go files"
	@echo "  deps             - Install required dependencies"

# Generate Go code from protobuf files
# Note: Run from api/proto directory with: make -C proto generate
generate:
	@echo "Generating Go code from protobuf files..."
	@cd .. && protoc \
		--proto_path=. \
		--proto_path=$(GOOGLEAPIS_PATH) \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. \
		--grpc-gateway_opt=paths=source_relative \
		proto/*.proto
	@echo "Go code generated successfully"

# Generate OpenAPI v2 (swagger.json) specs from proto files
generate-openapi:
	@echo "Generating OpenAPI specs from protobuf files..."
	@mkdir -p ../docs
	@cd .. && protoc \
		--proto_path=. \
		--proto_path=$(GOOGLEAPIS_PATH) \
		--openapiv2_out=../docs \
		--openapiv2_opt=logtostderr=true,json_names_for_fields=false \
		proto/auth.proto \
		proto/alert.proto \
		proto/billing.proto \
		proto/dashboard.proto \
		proto/scheduler.proto \
		proto/maintenance.proto \
		proto/notification_template.proto \
		proto/integration.proto \
		proto/reporting/reporting.proto
	@echo "OpenAPI specs generated in docs/"

# Clean generated files
clean:
	@echo "Cleaning generated Go files..."
	@rm -f *.pb.go
	@rm -f *_grpc.pb.go
	@echo "Clean complete"

# Install required dependencies
deps:
	@echo "Installing required dependencies..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(GRPC_GATEWAY_VERSION)
	@echo "Dependencies installed successfully"
