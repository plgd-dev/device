#!/usr/bin/env bash

logbt --setup
logbt --test
logbt -- /usr/local/bin/cloud_server $@