#!/bin/bash

# Exit values
# 0 correct version installed
# 3 not installed
# 4 older than required minimum version
# 5 newer than required maximum version

if [ ! -x "$(command -v docker-compose)" ]; then
	exit 3
fi

MIN_VERSION_MAJOR=1
MIN_VERSION_MINOR=17
MAX_VERSION_MAJOR=1
MAX_VERSION_MINOR=23

VERSION="$(docker-compose -v | sed 's/^.* \([0-9][0-9]*\)\.\([0-9][0-9]*\)\.\([0-9][0-9]*\).*$/\1 \2 \3/')"
VERSION_MAJOR="$(echo $VERSION | cut -d' ' -f1)"
VERSION_MINOR="$(echo $VERSION | cut -d' ' -f2)"

if [ "$VERSION_MAJOR" -lt "$MIN_VERSION_MAJOR" ] ||
	[ "$VERSION_MAJOR" -eq "$MIN_VERSION_MAJOR" -a "$VERSION_MINOR" -lt "$MIN_VERSION_MINOR" ]; then
	exit 4
elif [ "$VERSION_MAJOR" -gt "$MAX_VERSION_MAJOR" ] ||
	[ "$VERSION_MAJOR" -eq "$MAX_VERSION_MAJOR" -a "$VERSION_MINOR" -gt "$MAX_VERSION_MINOR" ]; then
	exit 5
else
	exit 0
fi
