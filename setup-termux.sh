#!/data/data/com.termux/files/usr/bin/bash

[ ! -d data ] && mkdir -p data
[ ! -d server/files/avatars ] && mkdir -p server/files/avatars
[ ! -f data/auth ] && touch data/auth
go build

sh='#!'
cdp="cd "
andp=" && ./wx && sleep(4)"
touch wx.sh
echo $sh$SHELL>wx.sh
echo $cdp$(pwd)$andp>>wx.sh
chmod ug+x wx.sh

echo "Please edit file 'config/conf.ini' to define path to your certificates"
