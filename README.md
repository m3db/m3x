Migration Warning
=================
This repository has been migrated to github.com/m3db/m3. It's contents can be found at github.com/m3db/m3/src/x. Follow along there for updates. This repository is marked archived, and will no longer receive any updates.

# M3X [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

Common utility code shared by all M3DB components.

<hr>

This project is released under the [Apache License, Version 2.0](LICENSE).

[doc-img]: https://godoc.org/github.com/m3db/m3x?status.svg
[doc]: https://godoc.org/github.com/m3db/m3x
[ci-img]: https://travis-ci.org/m3db/m3x.svg?branch=master
[ci]: https://travis-ci.org/m3db/m3x
[cov-img]: https://coveralls.io/repos/m3db/m3x/badge.svg?branch=master&service=github
[cov]: https://coveralls.io/github/m3db/m3x?branch=master

# Development

## Setup

1. Clone the repo into your $GOPATH
2. Run `git submodule update --init --recursive`
3. Run `glide install -v` - [Install Glide first if you don't have it](https://github.com/Masterminds/glide)
4. Run `make test` and make sure everything passes
5. Write new code and tests
