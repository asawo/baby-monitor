-include .env

PI_USER ?= pi
PI_HOST ?= monitor.local
PI      ?= $(PI_USER)@$(PI_HOST)
REMOTE  ?= /home/$(PI_USER)/monitor
PI_IP   ?= 192.168.1.100

.PHONY: build deploy sync install start stop status logs logs-http setup

setup:
	@sed 's/YOUR_PI_IP/$(PI_IP)/' mediamtx.yml.example > mediamtx.yml

build:
	GOOS=linux GOARCH=arm64 go build -o bin/monitor .

sync: setup
	rsync -av --exclude bin/ --exclude mediamtx ./ $(PI):$(REMOTE)/

deploy: build sync
	scp bin/monitor $(PI):/tmp/monitor-new
	ssh $(PI) "mv /tmp/monitor-new $(REMOTE)/monitor && chmod +x $(REMOTE)/monitor"

install: sync
	ssh $(PI) "$(REMOTE)/scripts/install.sh"

start:
	ssh $(PI) "$(REMOTE)/scripts/monitor.sh start" < /dev/null

stop:
	ssh $(PI) "$(REMOTE)/scripts/monitor.sh stop"

status:
	ssh $(PI) "$(REMOTE)/scripts/monitor.sh status"

logs:
	ssh $(PI) "tail -f $(REMOTE)/ffmpeg.log"

logs-http:
	ssh $(PI) "tail -f $(REMOTE)/monitor.log"
