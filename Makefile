.PHONY: build install test clean init run serve chat tg qq napcat weixin

HOME_DIR := $(CURDIR)/.lh-home

ifeq ($(OS),Windows_NT)
LH_BIN := .\lh.exe
ifeq ($(wildcard lh.exe),lh.exe)
LH_CMD := $(LH_BIN)
else
LH_CMD := go run ./cmd/la
endif
RUN_WITH_HOME = set "HOME=$(HOME_DIR)" &&
MODULE_TEST_ENV = set "GOWORK=off" &&
REMOVE_LH = if exist lh del /q lh & if exist lh.exe del /q lh.exe
else
LH_CMD := go run ./cmd/la
RUN_WITH_HOME = HOME="$(HOME_DIR)"
MODULE_TEST_ENV = GOWORK=off
REMOVE_LH = rm -f lh lh.exe
endif

build:
	go build -o lh ./cmd/la

install: build
	go install ./cmd/la

test:
	cd agent && $(MODULE_TEST_ENV) go test ./...
	go test ./...

init:
	$(RUN_WITH_HOME) $(LH_CMD) init

run: serve

serve:
	$(RUN_WITH_HOME) $(LH_CMD) serve

chat:
	$(RUN_WITH_HOME) $(LH_CMD) chat

tg:
	$(RUN_WITH_HOME) $(LH_CMD) msg-gateway start --platform telegram

qq:
	$(RUN_WITH_HOME) $(LH_CMD) msg-gateway start --platform qqofficial

napcat:
	$(RUN_WITH_HOME) $(LH_CMD) msg-gateway start --platform napcat

weixin:
	$(RUN_WITH_HOME) $(LH_CMD) msg-gateway start --platform weixin

clean:
	$(REMOVE_LH)
