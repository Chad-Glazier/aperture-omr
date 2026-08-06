go test \
	./internal/fs \
	./internal/server/handler \
	./internal/server/middleware \
	./internal/marker \
	./internal/pdf \
	./internal/scanner \
	./internal/sys \
    -coverprofile="coverage.out"
go tool cover -func="coverage.out"
rm coverage.out
