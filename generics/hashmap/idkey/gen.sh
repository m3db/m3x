#!/bin/sh

set -x
set -e

PKG="github.com/cheekybits/genny"

which genny >/dev/null || (go get -u $PKG && go install $PKG)
cat ../map.go | grep -v nolint | genny -pkg idkey -out ./map.go gen "KeyType=ident.ID ValueType=Value"
