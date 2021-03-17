#!/usr/bin/env bash

configfile=/src/config.yaml

if [ $inner_port ]; then
  echo "port: ${inner_port}" > $configfile
else
  echo "port: :2131" > $configfile
fi

if [ $time_interval ]; then
  echo "refresh_time_interval: ${time_interval}" >> $configfile
else
  echo "refresh_time_interval: 15" >> $configfile
fi

if [ $token ]; then
  echo "sub_token:
  enable: true
  token: ${token}" >> $configfile
else
  echo "sub_token:
  enable: false
  token: " >> $configfile
fi

./cli