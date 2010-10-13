GV=6

mcbot: mcserver
	$(GV)g mcbot.go items.go
	$(GV)l -o mc-bot mcbot.$(GV)

mcserver:
	$(GV)g -o mcserver.$(GV) server.go

clean:
	-rm *.$(GV) *~ 2> /dev/null 