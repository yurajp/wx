#!/bin/bash

[ ! -d data ] && mkdir -p data
[ ! -d server/files/avatars ] && mkdir -p server/files/avatars
[ ! -d server/files/records ] && mkdir -p server/files/records
[ ! -f data/regitry ] && touch data/registry
echo "{}">>registry


go build

sh='#!'
cdp="cd "
andp=" && ./wx && sleep(4)"
touch wx.sh
echo $sh$SHELL>wx.sh
echo $cdp$(pwd)$andp>>wx.sh
chmod ug+x wx.sh

echo "Please edit file 'config/conf.ini' to define path to your certificates"