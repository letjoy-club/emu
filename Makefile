build:
	CGO_ENABLED=0 go build

linux-server:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o emu

start:
	./emu

web:
	cd web && yarn build
	rm -rf static
	mv web/build/ static

.PHONY: build web
