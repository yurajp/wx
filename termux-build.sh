#!/data/data/com.termux/files/usr/bin/bash

[ ! -d data ] && mkdir -p data
[ ! -d server/files/avatars ] && mkdir -p server/files/avatars
[ ! -f data/auth ] && touch data/auth
go build

echo "You should edit the configuration file 'config/conf.ini' to define path to your certificates"

