# ksid: Go library for fast 63 bits unique IDs

[![Go Reference](https://pkg.go.dev/badge/github.com/maruel/ksid/.svg)](https://pkg.go.dev/github.com/maruel/ksid/)
[![codecov](https://codecov.io/gh/maruel/ksid/graph/badge.svg?token=DV0OVNS52P)](https://codecov.io/gh/maruel/ksid)

## Algorithm

I (Marc-Antoine Ruel) invented this algorithm in 2014 while working on the Chromium infrastructure. This was
primarily designed so I could generate unique IDs that would fit 63 bits integers that I could use on the
database.

The algorithm allows for 32000 unique IDs per microsecond, and has a lifetime of 86 years. Sharing across
servers is supported, as long as each server has a unique shard index and knows the total number of shards.
