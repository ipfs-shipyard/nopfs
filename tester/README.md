# Compact denylist format blocker validator

This module provides a test suite checks that the rules present in test.deny
are correctly understood and processed by a Blocker implementation.

The module includes its own `go.mod` to reduce dependency requirements to a
minimum. It is meant to be potentially used by other implementations as
needed.
