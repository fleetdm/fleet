# Simulate slow network

The following guide allows the developer/tester to simulate a "slow" connection from Fleet to Redis/MySQL.
(It could be used in a similar way to simulate a slow Fleet server too.)

The guide assumes you'll build and run the Fleet server locally with the `make fleet` and `./build/fleet serve` commands.

(Has been tested on macOS only.)

## 0. Edit docker-compose.yml

Add the following service to the main `docker-compose.yml`:
```yml
  toxiproxy:
    image: shopify/toxiproxy
    ports:
      - "22220:22220"
      - "8474:8474"
```

## 1. Start services

```sh
docker-compose up
```

## 2. Build Fleet

```sh
make fleet
```

## 3. Create a new proxy

The following command will create a "slow" proxy to the MySQL server that listens on 22220.

```sh
curl -s -XPOST -d '{"name" : "mysql", "listen" : "toxiproxy:22220", "upstream" : "mysql:3306"}' http://localhost:8474/proxies
{"name":"mysql","listen":"172.30.0.9:22220","upstream":"mysql:3306","enabled":true,"toxics":[]}%
```

## 4. Run fleet

Run fleet as usual but connect to MySQL via the proxy created in step (3):
```sh
./build/fleet serve --dev --dev_license --logging_debug --mysql_address localhost:22220 2>&1 | tee ~/fleet.txt
```

## 5. Configure proxy with latency/jitter

Configure 1 second of latency with 500ms of jitter in all DB requests:
```sh
curl -s -XPOST -d '{"type" : "latency", "attributes" : {"latency" : 1000, "jitter": 500}}' http://localhost:8474/proxies/mysql/toxics
{"attributes":{"latency":5000,"jitter":0},"name":"latency_downstream","type":"latency","stream":"downstream","toxicity":1}%
```

<meta name="pageOrderInSection" value="1400">
<meta name="description" value="A guide for simulating slow connections from a local Fleet server to a Redis or MySQL database">
