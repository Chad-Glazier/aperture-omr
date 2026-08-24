go test `
	--count=1 `
    -coverprofile="coverage.out" `
	.\...
go tool cover -func="coverage.out"
rm coverage.out
