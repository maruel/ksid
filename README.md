# ksid: Go library for fast 63 bits unique IDs

[![Go Reference](https://pkg.go.dev/badge/github.com/maruel/ksid/.svg)](https://pkg.go.dev/github.com/maruel/ksid/)
[![codecov](https://codecov.io/gh/maruel/ksid/graph/badge.svg?token=DV0OVNS52P)](https://codecov.io/gh/maruel/ksid)

## Properties

- 63 bits unique IDs
- Native base32 encoding, at most 13 characters
- Both numerical and string forms are k-sortable
- Shardable across up to 32768 servers
- Valid until 2115, or 2204 using the 64th bit

Performance

- ksid.ID is 50% smaller (8 bytes) than UUIDs (16 bytes) in binary form.
- ksid.ID string is at most 13 characters, 33% the size of UUIDs (36 characters).
- ksid.ID are faster to generate, encode and decode than UUIDs.
- Recommended generation throughput: <=1000 IDs per microsecond, <=100 shards.
- Maximium generation throughput: 32768 ID/µs and 3.27B ID/s.

Comparison with `google/uuid`:
- Generation: ksid (~33 ns) is at least ~7.5x faster than both uuid v4 (~250 ns) and uuid v7 (~290 ns).
- Encoding: ksid (~24 ns) is ~2.7x faster than uuid (~65 ns).
- Decoding: ksid (~8 ns) is ~1.8x faster than uuid (~15 ns).

```
$ go test -bench=.
goos: linux
goarch: amd64
pkg: github.com/maruel/ksid
cpu: 13th Gen Intel(R) Core(TM) i7-13700
BenchmarkID/ksid-gen-24      36123418   33.15 ns/op   0 B/op  0 allocs/op
BenchmarkID/uuidv4-gen-24     4740570  248.8  ns/op  16 B/op  1 allocs/op
BenchmarkID/uuidv7-gen-24     4138654  288.5  ns/op  16 B/op  1 allocs/op
BenchmarkID/ksid-encode-24   95906893   24.37 ns/op  16 B/op  1 allocs/op
BenchmarkID/uuid-encode-24   23047898   64.62 ns/op  48 B/op  1 allocs/op
BenchmarkID/ksid-decode-24  141703341    8.42 ns/op   0 B/op  0 allocs/op
BenchmarkID/uuid-decode-24   78804513   14.89 ns/op   0 B/op  0 allocs/op
```

## Algorithm

I (Marc-Antoine Ruel) invented this algorithm in 2014 while working on the Chromium infrastructure. I designed
the algorithm so I could generate unique IDs that would fit 63 bits integers to use as database keys.

The library has 100% unit testing code coverage.
