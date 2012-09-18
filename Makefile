mcbot:
	go build -o mc-bot
 
prereq:
	go get 'github.com/ckolbeck/mcserver'
	go get 'github.com/ckolbeck/ircbot'