SHELL := /usr/bin/env bash

.PHONY: check ci config inventory repository-check workflows

config:
	golib config validate

inventory:
	golib inventory

repository-check:
	golib repository check

workflows:
	golib workflows check

check:
	golib check --all

ci: config inventory repository-check workflows check
