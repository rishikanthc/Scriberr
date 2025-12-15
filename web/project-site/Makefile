.PHONY: build dev clean install

install:
	npm install

generate-api:
	cd ../.. && swag init -g server/main.go -o api-docs --dir cmd,internal

build: generate-api
	npm run build

dev: generate-api
	npm run dev

clean:
	rm -rf dist node_modules
