build:
	CGO_ENABLED=0 go build

start:
	./emu

web:
	cd web && yarn build
	rm -rf static
	mv web/build/ static

.PHONY: build web
