BINARY  := ob
GOFLAGS ?=
PREFIX  ?= /usr/local
DESTDIR ?=

.PHONY: all build install uninstall clean fmt vet run

all: build

build:
	go build $(GOFLAGS) -o $(BINARY) .

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)

fmt:
	gofmt -l -w .

vet:
	go vet ./...

run: build
	./$(BINARY) $(ARGS)
