build:
	go build -v

fmt:
	find . -name "*.go" -not -path "./vendor/*" | xargs gofmt -w -s

docker:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a --ldflags "-s"
	docker build --rm -t docker.io/perflab/daily .

clean:
	rm -fr daily
