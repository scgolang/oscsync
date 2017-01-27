# oscsync

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

Tempo is expressed as a 64-bit float that represents beats per minute. A beat is a quarter note and we will use those two terms interchangeably in this document.

### Bar

A bar is 4 quarter notes (or beats) at a given tempo.

### Node

A program running on a computer. A node can either be a master or slave.

#### Master

Responsible for broadcasting a sync message under two conditions:

* Tempo changes.
* A new bar begins.

#### Slave

Responsible for providing process synchronization according to messages it receives from a Master.

Slaves

## Acknowledgements

The ideas expressed by RBD in [this paper](http://opensoundcontrol.org/files/dannenberg-clocksync.pdf) on synchronization had a large impact on the design of OSC Sync.
