go test `
	--count=1 `
	./internal/fs `
	./internal/server/handler `
	./internal/server/mw `
	./internal/server/dto `
	./internal/marker `
	./internal/pdf `
	./internal/scanner `
	./internal/sys `
    -coverprofile="coverage.out"
go tool cover -func="coverage.out"
rm coverage.out
