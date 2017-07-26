#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "${BASH_SOURCE%/*}/common.bash"

r=$(mktemp -d)
t=$(mktemp -d)

echo "Cloning git@github.com:myitcv/gopherize.me_site.git into $r"

echo ""

echo "Copying..."

(
	cd $t
	wget --quiet --mirror http://localhost:8080/myitcv.io/gopherize.me/client/
)
cp -rp $t/localhost:8080/myitcv.io/gopherize.me/client/ $r/gopherize.me_site/

du -sh $r/gopherize.me_site

cp -rp artwork $r/gopherize.me_site/

echo ""

cd $r/gopherize.me_site
git init
git config hooks.stopbinaries false
touch .nojekyll

git add -A
git commit -am "Examples update at $(date)"

git remote add origin git@github.com:myitcv/gopherize.me_site.git

git push -f
