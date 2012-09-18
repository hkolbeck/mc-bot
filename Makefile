mcbot:
	go build -o mc-bot
 
prereq:
	go get 'https://github.com/ckolbeck/mcserver'
	go get 'https://github.com/ckolbeck/ircbot'