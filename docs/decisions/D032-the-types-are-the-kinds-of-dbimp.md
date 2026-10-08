# D32. The types are the kinds of dbimp

Status: Decided. Amends D17. Replaces D26. Amended by D35.

Ken decided on 2026-10-08 that each column has the Go type that fits the type
that Cassandra names for it, by the kinds of the types document of `dbimp`
(dbimp D135, dbimp D137 and dbimp D138).

- A decimal is an `*apd.Decimal`. A varint is a `*big.Int`. Both were a string.
- A date is a `dbimp.Date`, a time is a `dbimp.LocalTime`, and a duration is a
  `dbimp.Interval`. They were a `time.Time`, a `time.Duration` and a
  `gocql.Duration`.
- An inet is a `netip.Addr`. It was a string.
- A list, a set and a tuple are a `[]any` of the Go types of their elements. A
  map with a text key and a user defined type are a `map[string]any`. A vector of
  numbers is a `dbimp.Vector[T]`. They were the Go types that gocql chooses.
- The integer types, the floats, the text types, the blob, the boolean, the
  timestamp and the UUID keep their Go types. A UUID is the standard `uuid.UUID`
  (D25).
- A map with a key that is not text stays the map that gocql chooses, and a
  varint is the kind `big integer` of `dbimp` although the kind names 128 and 256
  bits (open questions 14 and 15).
- The driver binds the same types as arguments. gocql marshals a decimal from an
  `*inf.Dec`, so the driver writes the wire form of an `*apd.Decimal` itself.

The type table is in [CASSANDRA.md](../CASSANDRA.md), and a test writes it from
the code.
