include $(GOROOT)/src/Make.inc

mcbot: mcserver
	$(GC) mcbot.go items.go
	$(LD) -o mc-bot mcbot.$(GV)

mcserver:
	cd server
	make
	make install

test:
	$(GC) -o test.6 config.go driver.go
	$(LD) -o test test.6