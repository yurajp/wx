#!/data/data/com.termux/files/usr/bin/bash

mkdir -p data
mkdir -p server/files/avatars
go build

echo "You should edit the configuration file 'config/conf.ini' to define path to your certificates"

