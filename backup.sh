#!/usr/bin/env bash
#Copyright 2010 Cory Kolbeck <ckolbeck@gmail.com>.
#So long as this notice remains in place, you are welcome 
#to do whatever you like to or with this code.  This code is 
#provided 'As-Is' with no warrenty expressed or implied. 
#If you like it, and we happen to meet, buy me a beer sometime

tar -czf mcServerBackups/$1 banned-ips.txt server.log server.properties banned-players.txt ops.txt world
