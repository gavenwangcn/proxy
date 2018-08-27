#!/bin/bash

start_proxy() {
    ./proxy --config ./config.json  --log-file  ./proxy.log
}

start_proxy
