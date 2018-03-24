#!/bin/sh

set -x
set -e

PKG="github.com/cheekybits/genny"

which genny >/dev/null || (go get -u $PKG && go install $PKG)
cat ../map.go | genny -pkg idkey -out ./map.go gen "KeyType=ident.ID ValueType=Value"
