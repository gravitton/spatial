<div align="center" width="100%">

<a href="https://github.com/gravitton">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/gravitton/spatial/refs/heads/main/docs/images/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/gravitton/spatial/refs/heads/main/docs/images/logo-light.svg">
  <img alt="Gravitton spatial" src="https://raw.githubusercontent.com/gravitton/spatial/refs/heads/main/docs/images/logo-light.svg" width="300">
</picture>
</a>

[![Latest Stable Version][ico-release]][link-release]
[![Build Status][ico-workflow]][link-workflow]
[![Coverage Status][ico-coverage]][link-coverage]
[![Go Dev Reference][ico-go-dev-reference]][link-go-dev-reference]
[![Software License][ico-license]][link-licence]

Broad-phase spatial indexes for game development

<hr>

</div>


## Features

- **Generic** – any comparable key, any [`geometry`][link-geometry] number type for coordinates.
- **Broad phase only** – an index answers which keys may touch a region; the shapes decide.
- **Spatial hash** – `Hash`, a uniform grid of cells over an unbounded plane.

The rules behind them are in the [package documentation][link-go-dev-reference].

## Installation

```shell
go get github.com/gravitton/spatial
```

## Usage

```go
import (
	geom "github.com/gravitton/geometry"
	"github.com/gravitton/spatial"
)
```

```go
index := spatial.NewHash[int, float64](64)
```

## Credits

- [Tomáš Novotný](https://github.com/tomas-novotny)
- [All Contributors][link-contributors]

## License

The MIT License (MIT). Please see [License File][link-licence] for more information.


[ico-license]:              https://img.shields.io/github/license/gravitton/spatial.svg?style=flat-square&color=blue
[ico-workflow]:             https://img.shields.io/github/actions/workflow/status/gravitton/spatial/main.yml?branch=main&style=flat-square
[ico-release]:              https://img.shields.io/github/v/release/gravitton/spatial?style=flat-square&color=blue
[ico-go-dev-reference]:     https://img.shields.io/badge/go.dev-reference-blue?style=flat-square
[ico-coverage]:             https://img.shields.io/coverallsCoverage/github/gravitton/spatial?style=flat-square

[link-author]:              https://github.com/gravitton
[link-release]:             https://github.com/gravitton/spatial/releases
[link-contributors]:        https://github.com/gravitton/spatial/contributors
[link-licence]:             ./LICENSE.md
[link-changelog]:           ./CHANGELOG.md
[link-workflow]:            https://github.com/gravitton/spatial/actions
[link-go-dev-reference]:    https://pkg.go.dev/github.com/gravitton/spatial
[link-geometry]:            https://github.com/gravitton/geometry
[link-coverage]:            https://coveralls.io/github/gravitton/spatial
