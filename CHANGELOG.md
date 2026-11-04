# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).


## [Unreleased](https://github.com/gravitton/spatial/commits/main)
### Added
- `Hash`, a uniform spatial hash over an unbounded plane: `NewHash`, `CellSize`, `Set`, `Get`, `Delete`, `Len`, `Clear` and `All`, with `ErrInvalidCellSize` wrapped by the panic of `NewHash`
- `Hash.QueryBox` and `Hash.QueryPoint`, with their `Append` forms
- `Hash.QuerySegment` and `Hash.QueryRay` as `iter.Seq` that can stop early, with their `Append` forms
- `Hash.Pairs` and `Hash.AppendPairs`, every pair of touching keys once, and the `Pair` they return

### Changed
- Require `geometry` v1.18.0, and `assert` v1.7.0 for the tests
- CI runs the shared workflow of `gravitton/.github`, adding arm64 and the fused multiply-add check, and can be started by hand


