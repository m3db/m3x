#!/bin/sh

set -x
set -e

PKG="github.com/cheekybits/genny"

which genny >/dev/null || (go get -u $PKG && go install $PKG)
cat ../map.go | genny -pkg byteskey -out ./map.go gen "KeyType=[]byte ValueType=Value"
