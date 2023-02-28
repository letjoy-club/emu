build:
	go build

web:
	cd web && yarn build
	rm -rf static
	mv web/build/ static

.PHONY: build web
