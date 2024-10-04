#!/bin/bash -e
set -x #echo on

CGO_ENABLED=0 go build -x -o dummy-ads .

VERSION=`./dummy-ads -version | cut -d' ' -f3`
name=dummy-ads-${VERSION}
cp dummy-ads ${name}
tar cvzf ${name}.tar.gz ${name}
rm ${name}
