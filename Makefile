COMMIT_MSG_HOOK = '\#!/bin/bash\nMSG_FILE=$$1\ncz check --allow-abort --commit-msg-file $$MSG_FILE'
.PHONY: build test

setup-dev-env:
	pip install pre-commit==3.8.0 commitizen==3.29.0 python-semantic-release==9.8.8
	pre-commit install
	echo $(COMMIT_MSG_HOOK) > .git/hooks/commit-msg
	chmod +x .git/hooks/commit-msg

test:
	go install github.com/onsi/ginkgo/v2/ginkgo
	ginkgo -cover --junit-report=report.xml ./...

generate-localhost-ca:
	openssl req -x509 -newkey rsa:4096 -keyout ca-key.pem -out ca-cert.pem -nodes -subj "/C=TW/O=example.com/OU=example/CN=localhost/emailAddress=example@example.com"

build:
	go mod tidy
	go build -a -o build/mutating-webhook cmd/main.go


ifeq (run, $(firstword $(MAKECMDGOALS)))
  runargs := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
  $(eval $(runargs):;@true)
endif

run: build
	./build/mutating-webhook $(runargs)
