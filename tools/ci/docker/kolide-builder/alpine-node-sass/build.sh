#!/bin/sh
REV="${NODE_SASS_VERSION:-master}"
echo "Building node-sass $REV"

# clone and build the node-sass version
git clone --recursive https://github.com/sass/node-sass.git
cd node-sass
git checkout $REV
git submodule update --init --recursive
npm install
node scripts/build -f 

# rename and move /node-sass/vendor/linux-x64-*/binding.node
# to /build/$REV/linux-x64-version_binding.node
mkdir -p /build
cd vendor
for file in $(find . -type f); do
	RENAME=$(echo "$file" | awk 'BEGIN { FS = "/" } ; {print $2"_"$3}')
	cp $file "/build/$RENAME"
done
