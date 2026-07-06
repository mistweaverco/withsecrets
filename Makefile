BIN_NAME=ws

build:
	./scripts/build.sh

gui-build:
	./scripts/build-gui.sh

test:
	./scripts/test.sh

test-coverage:
	MODE=coverage ./scripts/test.sh

release:
	./scripts/release.sh
