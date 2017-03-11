# oscsync

Synchronize programs over OSC.

[Install](#install)
[Getting Started](#getting-started)
[API](#api)

## Install

```
go get github.com/scgolang/oscsync
```

## Getting Started

Run the master:

```
oscsync serve
```

Connect a slave that just prints the pulses coming from the master:

```
oscsync pulses
```

## API

### Pulse

`/sync/pulse f:tempo i:position`

A pulse tells clients what position the master is at.
The position is interpreted as 24ppqn at the given tempo.

### Add Slave

`/sync/slave/add s:host i:port`

Add a slave who is listening at the given host:port.

### Remove Slave

`/sync/slave/remove s:host i:port`

Remove the slave who is listening at the given host:port.
