.PHONY: build snapshot release check clean

build:
	goreleaser build --snapshot --clean --single-target

snapshot:
	goreleaser build --snapshot --clean

release:
	goreleaser release --clean

check:
	goreleaser check

clean:
	rm -rf dist/
