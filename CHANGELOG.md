# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).


## [Unreleased](https://github.com/gravitton/spatial/compare/v0.1.0...main)

## [v0.1.0](https://github.com/gravitton/spatial/commits/v0.1.0) (2026-11-06)
### Added
- `Hash`, a uniform spatial hash over an unbounded plane on `geometry` v1.18.0: `NewHash`, `CellSize`, `Set`, `Get`, `Delete`, `Len`, `Clear` and `All`, with `ErrInvalidCellSize` wrapped by the panic of `NewHash`
- `Hash.QueryBox` and `Hash.QueryPoint`, with their `Append` forms
- `Hash.QuerySegment` and `Hash.QueryRay` as `iter.Seq` that can stop early, with their `Append` forms
- `Hash.Pairs` and `Hash.AppendPairs`, every pair of touching keys once, and the `Pair` they return


