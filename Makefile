SHELL = /bin/sh

.PHONY: default linux windows clean clobber install docker_usb docker_usb_windows docker_usb_linux tag

RADIO_SRC = *.go
CODEPLUG_SRC = ../codeplug/*.go
DFU_SRC = ../dfu/*.go
STDFU_SRC = ../stdfu/*.go
USERDB_SRC = ../userdb/*.go
DEBUG_SRC = ../debug/*.go
RADIO_SRCS =  $(RADIO_SRC) $(CODEPLUG_SRC) $(DFU_SRC) $(STDFU_SRC) $(USERDB_SRC)
VERSION = $(shell sed -n '/version =/{s/^[^"]*"//;s/".*//p;q}' <version.go)

default: linux windows

linux: dmrRadio

windows: dmrRadio.exe

install: dmrRadio-$(VERSION).tar.xz dmrRadio-$(VERSION)-installer.exe dmrRadio-changelog.txt

dmrRadio-$(VERSION)-installer.exe: dmrRadio.exe dmrRadio.nsi dll/*.dll
	makensis -DVERSION=$(VERSION) dmrRadio.nsi

dmrRadio: $(RADIO_SRCS)
	go build

dmrRadio.exe: $(RADIO_SRCS)
	GOOS=windows GOARCH=386 go build

dmrRadio-$(VERSION).tar.xz: dmrRadio
	rm -rf dmrRadio-$(VERSION)
	mkdir -p dmrRadio-$(VERSION)
	cp -al dmrRadio dmrRadio-$(VERSION)
	tar cJf dmrRadio-$(VERSION).tar.xz dmrRadio-$(VERSION)
	rm -rf dmrRadio-$(VERSION)

FORCE:

dmrRadio-changelog.txt: FORCE
	sh generateChangelog >dmrRadio-changelog.txt

clean:
	rm -f dmrRadio dmrRadio.exe

clobber: clean
	rm -f dmrRadio-*.tar.xz dmrRadio-*-installer.exe dmrRadio-changelog.txt
