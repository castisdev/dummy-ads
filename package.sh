#!/bin/bash -e
set -x #echo on

sudo docker run --rm -v $(pwd):$(pwd) -w $(pwd) ghcr.io/castisdev/centos7:1.17 go build -buildvcs=false -x -o dummy-ads .
sudo chown -R $(whoami):$(whoami) dummy-ads

VERSION=`./dummy-ads -version | cut -d' ' -f3`
name=dummy-ads-${VERSION}
cp dummy-ads ${name}
tar cvzf ${name}.tar.gz ${name}
rm ${name}
