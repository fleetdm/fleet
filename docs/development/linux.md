## Setting up a Linux Development Environment

### Install some dependencies

`sudo apt-get install xzip gyp libjs-underscore libuv1-dev dep11-tools deps-tools-cli`

### Create a temp directory, download and place the `node` and `golang` bins 

```
mkdir tmp
cd tmp
```

#### install `node` and `yarn`

```
wget https://nodejs.org/dist/v9.4.0/node-v9.4.0-linux-x64.tar.xz
xz -d node-v9.4.0-linux-x64.tar.xz
tar -xf node-v9.4.0-linux-x64.tar
sudo cp -rf node-v9.4.0-linux-x64/bin /usr/local/
sudo cp -rf node-v9.4.0-linux-x64/include /usr/local
sudo cp -rf node-v9.4.0-linux-x64/lib /usr/local
sudo cp -rf node-v9.4.0-linux-x64/share /usr/local
npm install -g yarn
```

#### install `go`

```
wget https://dl.google.com/go/go1.9.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.9.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin:~/go/bin/
```

#### clean-up temp directory

```
cd ..
rm -rf tmp
```

### Clone and build depenencies

```
git clone https://github.com/fleetdm/fleet.git
cd fleet
make deps
make generate
make build
sudo cp build/fleet /usr/bin/fleet
```
