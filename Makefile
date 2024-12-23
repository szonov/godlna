#!/usr/bin/make
# Makefile readme (ru): <http://linux.yaroslavl.ru/docs/prog/gnu_make_3-79_russian_manual.html>
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

.PHONY : help work
.DEFAULT_GOAL : help

# This will output the help for each task. thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[32m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

rsyno: ## Build and send to first synology box
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./godlna cmd/godlna/godlna.go
	scp -O godlna rsyno:godlna/
	scp -O ./scripts/synology-install.sh rsyno:godlna/
	ssh rsyno chmod 755 godlna/godlna
	ssh rsyno chmod 755 godlna/synology-install.sh
	rm ./godlna	