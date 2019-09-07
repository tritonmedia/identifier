
# identifier

[![goreport](https://goreportcard.com/badge/github.com/tritonmedia/identifier)](https://goreportcard.com/report/github.com/tritonmedia/identifier)

Identify your media regardless of the metadata provider.

## What is this?

Identifier identifies media using a metadata provider such as; TVDB, Kitsu, IMDB, and etc, and stores that information in Postgres. Information, such as; Series Details, Episodes, Images, and etc.

Identifier recieves jobs over AMQP to identify media then stores that inside PostgreSQL.

## Building

This uses libvips which requires vips to be installed.

### Mac OSX

```bash
$ brew install pkg-config vips libffi
$ export PKG_CONFIG_PATH="/usr/local/opt/libffi/lib/pkgconfig"
... regular commands
```

## License

Apache-2.0
