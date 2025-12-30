.PHONY: build dev clean install

install:
	npm install

generate-api:
	cd ../.. && swag init -g server/main.go -o web/project-site/public/api --outputTypes json --dir cmd,internal

build: generate-api
	npm run build

dev: generate-api
	npm run dev

clean:
	rm -rf dist node_modules
