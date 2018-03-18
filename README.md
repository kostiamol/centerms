[![Go Report Card](https://goreportcard.com/badge/github.com/kostiamol/centerms)](https://goreportcard.com/report/github.com/kostiamol/centerms)
[![Coverage Status](https://coveralls.io/repos/github/kostiamol/centerms/badge.svg?branch=master)](https://coveralls.io/github/kostiamol/centerms?branch=master)
[![Build Status](https://travis-ci.org/kostiamol/centerms.svg?branch=master)](https://travis-ci.org/kostiamol/centerms)

# centerms
The project implements a server-side management automation for a home.

## Quickstart
1. [Install and run the NATS server](https://github.com/nats-io/gnatsd#quickstart)
2. [Install and run the Redis server](https://redis.io/topics/quickstart#installing-redis)
3. [Install and run Consul](https://www.consul.io/intro/getting-started/install.html)
4. Optional: Install and run Prometheus + Grafana
5. Download and install the centerms:

```bash
go get github.com/kostiamol/centerms
```

6. Compile and run the centerms:

```bash
cd $GOPATH/src/github.com/kostiamol/centerms/cmd/centerms
go build 
./centerms
```

7. For proper functioning of the system as a whole install and run the [dashboard](https://github.com/kostiamol/dashboard-ui) and some of the "devices":
    - [fridgems](https://github.com/kostiamol/fridgems)
