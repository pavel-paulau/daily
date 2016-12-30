build:
	go build -v

fmt:
	find . -name "*.go" -not -path "./vendor/*" | xargs gofmt -w -s

clean:
	rm -fr daily
