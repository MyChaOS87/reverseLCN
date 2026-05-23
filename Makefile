.PHONY: all test lint tidy tools-update tools-tidy upgrade-direct-dependencies build github-actions-test

all: tidy build lint test

build:
	go build ./...

test:
	go test -v -race ./...

lint:
	go tool -modfile=tools/golangci-lint/go.mod golangci-lint run ./...

tidy:
	go mod tidy
	git diff --exit-code go.mod go.sum

github-actions-test:
	go tool -modfile=tools/act/go.mod act -j test-and-build --artifact-server-path ~/.act/artifacts




tools-update:
	for d in tools/* ; do \
		if [ -d $$d ] && [ -f $$d/go.mod ] ; then \
			echo "Updating $$d" ; \
			cd $$d ; \
			go get -u $$(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all) ; \
			go mod tidy ; \
			cd - >/dev/null; \
		fi ; \
	done

tools-tidy:
	for d in tools/* ; do \
		if [ -d $$d ] && [ -f $$d/go.mod ] ; then \
			echo "tidy $$d" ; \
			cd $$d ; \
			go mod tidy ; \
			cd - >/dev/null; \
		fi ; \
	done

upgrade-direct-dependencies:
	go get -u ./...
	go mod tidy
