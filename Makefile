BIN_NAME=ws

build:
	./scripts/build.sh

test:
	./scripts/test.sh

test-coverage:
	MODE=coverage ./scripts/test.sh

release:
	./scripts/release.sh
