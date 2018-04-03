# Mining Monitor

This program will periodically poll your mining rigs for statistics and attempt to reboot/power cycle if things are going wrong.

# Setup

This repository contains the library and a sample `main.go` of its usage to control a claymore miner using the remote management API.

Currently the sample `main.go` assumes the following setup:

1. [Claymore Miner](https://bitcointalk.org/index.php?topic=1433925.0) running version >= 10.2, with remote management enabled and a strong password set.
2. Mining rig connected to a [TPLink HS110](https://www.tp-link.com/us/products/details/cat-5516_HS110.html).

It is **strongly** recommended to run this program on the same local network as your miner. Exposing the claymore management port outside your local network is not recommended.

Example run arguments:

```go
./main -logtostderr \
    -claymore-address 192.168.0.10 \
    -claymore-password password \
    -hs110plug-ip 192.168.0.17 \
    -email <Gmail Address> \
    -email-host smtp.gmail.com \
    -email-password <Google App Password> \
    -hash-threshold "<18000" \
    -fan-threshold 70 \
    -power-cycle-only \
    -v=2
```

# Simple Architecture Diagram of Library and Usage

```
[Monitor]
    |- [EventService]
        |- [EmailService - GMailService]
    |- [Client - Claymore 11.0]
        |- [PowerService - HS110PowerService]
        |- [Threshold - HashThreshold]
        |- [Threshold - FanThreshold]
    |- [Client - Claymore 10.2]
        |- [PowerService - CustomSmartPlug]
        |- [Threshold - TemperatureThreshold]
        |- [Threshold - PowerThreshold]
```

Monitor is the top level service, each monitor will have an Event service to handle events from the clients being monitored. Each client can have its own set of thresholds (or shared) and its own power service.

# Customization

This project is a library, meant to enable developers to customize it to suite their needs. The `main.go` is a simple example on how to do so. The library consists of a few interfaces which should be implemented if you use something other than Claymore, or a HS110 Smart Plug. See the [Docs](https://godoc.org/github.com/mchestr/mining-monitor) for more info.

Currently to modify the `main.go` you are required to have go installed. However this project could be extended to read in configuration from a YAML file, but for now it suited my needs as I only have 1 mining rig. If there is a desire for this feel free to open an issue.
