# Memsy 

A durable memcache replacement


## Overview

Memsy (prouncned mem-c as in memcached) is a wire compatible memcache replacement written in go, it is in memory but simultaneously backs to disk, if the process is restarted the disk cache is reloaded into memory.

Memsy also allows you to create a cluster of peers that will keep in sync with each other.  Each install needs to just reference the others, not mandatory to use

## Configuration

Memsy is configured on the command line

```
  -cachedir string
        Location directory for memsy disk db (default "/var/cache")
  -comport string
        TCP port number for communication (default "1180")
  -debug
        Debug settings
  -listen string
        Interface to listen on. Default to all addresses. (default "0.0.0.0")
  -peers string
        Comma separated list of servers to peer with
  -port int
        TCP port number to listen on (default: 11211)
  -syncinterval string
        How often to sync all records to other nodes (default "30m") Note: All records are synced as they come in, this covers old records, etc
  -threads int
        number of threads to use (default: # of cpus on server)
   
```
        
Note: the comport and port #'s must be the same on all peers for syncing to work

## Install

Copy the memsy binary to your server and run

```
./memsy --peers=1.2.3.4,4.3.2.1 --port=11212
```

## Building from source

cd to directory and then run "go build"

## FAQ

- Q: How does syncing work
- A: When a set request comes in it's batched in groups (using a rate limit algorithm) and copied to all peers configured (this reduces connection overhead to peers), every syncinterval all keys are sent to peer nodes and diff'd, what doesn't match is sent to the peer. This is also done on startup.

- Q: What is the protocol for syncing
- A: The key diffing protocl using tcp, once the list of keys is determined the keys/values are sent using memcache protocl itself
