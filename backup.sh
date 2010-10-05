#!/usr/bin/env bash

tar -czf mcServerBackups/$1 banned-ips.txt server.log server.properties banned-players.txt ops.txt world
