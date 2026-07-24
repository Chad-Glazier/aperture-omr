go test \
	./internal/fs \
	./internal/httpserver/handler \
	./internal/httpserver/middleware \
	./internal/marker \
	./internal/pdf \
	./internal/scanner \
    -coverprofile="coverage.out"
go tool cover -func="coverage.out"
rm coverage.out
