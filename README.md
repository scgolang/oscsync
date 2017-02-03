# oscsync

This document, as well as the reference implementation whose code lives next to this document, is a Work in Progress.

OSC Sync Protocol

## Problem

Music programs that are clock-driven (e.g. sequencers) need a way to synchronize to each other.

## Requirements

### Terminology

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this
document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

An implementation is not compliant if it fails to satisfy one or more
of the MUST or REQUIRED level requirements for the protocols it
implements. An implementation that satisfies all the MUST or REQUIRED
level and all the SHOULD level requirements for its protocols is said
to be "unconditionally compliant"; one that satisfies all the MUST
level requirements but not all the SHOULD level requirements for its
protocols is said to be "conditionally compliant."

### Implementation Requirements

* Cross-platform, minimal dependencies.
* Provides musically accurate timing (not necessarily sample accurate, it just needs to sound good > 99% of the time).
* Implemented as OSC over UDP (most programs that use OSC will expect this).
* Tolerant of temporary spikes in network latency as well as clock drift between systems.

## Solution

The OSC Sync Protocol claims to solve the stated problem while meeting the above requirements.

The rest of this document details the protocol.

## Protocol

The OSC Sync Protocol consists of the following primitives:

### Tempo

Tempo MUST be expressed as a 32-bit float that MUST represent beats per minute. A beat MUST be a quarter note and we will use those two terms interchangeably in this document.

### Bar

A bar MUST be treated as 4 quarter notes (or beats) at a given tempo.

### Pulse

A pulse is a subdivision of a quarter note such that there MUST be 24 pulses per quarter note (ppqn).

A point in time SHOULD be specified as **BAR** **BEAT** **PULSE**, where **BAR** >= 1, **BEAT** is in the interval [0, 3], and **PULSE** is in the interval [0, 23].

### Node

A node is a program running on a computer which can either be a Master or a Slave.

#### Master

The master MUST provide a counter that the current bar.

The counter MUST be broadcast to all registered slaves every time the master moves forward one bar.

#### Slave

A slave MUST position itself according to the master's counter.

A slave registers it's IP address and listening port with a master whenever it wishes to synchronize with the master. After that synchronization happens less frequently (once per bar) to conserve network bandwidth.

After a slave registers the MASTER must send it a message on the next pulse indicating the tempo and the exact pulse that the master is at. This is so the slave doesn't have to wait until the next bar to synchronize with the master.

## Methods

### Master

The master MUST provide the following methods:

| Address                                                    | Description
| ---------------------------------------------------------- | --------------------------------------
| /sync/tempo f:bpm                                          | Update the Node's tempo.
| /sync/slave/add s:address i:port                           | Used by slaves to register themselves with the Master.
| /sync/slave/remove s:address i:port                        | Used by slaves to deregister themselves with the Master.

The master MUST broadcast a sync message under any of the following conditions:

* Tempo changes.
* A new bar begins.

The master MAY also provide the following methods for monitoring/debugging:

| Address                                                                | Description
| ---------------------------------------------------------------------- | -----------------------------------------
| /sync/slave/list i:num_slaves [s:address i:port [s:address i:port]...] | Get the list of slaves.

### Slave

Responsible for synchronizing a process to a Master.

Part of this responsibility is that a slave SHOULD connect to a Master node when it's process starts.

Slaves MUST provide the following methods:

| Address                                         | Description
| ----------------------------------------------- | --------------------------------------
| /sync/tempo f:bpm                               | Update the Node's tempo.
| /sync/counter i:bar                             | Slave MUST ensure that it's internal clock is at `bar`.
| /sync/pulse f:bpm i:bar i:beat i:pulse          | Slave SHOULD use this message for fine-grained synchronization with the master.

Slaves MAY also implement the following methods:

| Address                                         | Description
| ----------------------------------------------- | --------------------------------------
| /sync/register s:address i:port                 | The slave MUST synchronize to the master at address:port

## Acknowledgements

The ideas expressed by Roger Dannenberg in [this paper](http://opensoundcontrol.org/files/dannenberg-clocksync.pdf) on synchronization had a large impact on the design of OSC Sync.
