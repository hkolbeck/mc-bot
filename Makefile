include $(GOROOT)/src/Make.inc

mcbot: 
	$(GC) mcbot.go config.go commands.go io.go
	$(LD) -o mc-bot mcbot.$(GV)

mcserver:
	cd server
	make
	make install
	cd -
test:
	$(GC) -o test.6 config.go driver.go
	$(LD) -o test test.6

clean:
	rm *~ *.$(O) 