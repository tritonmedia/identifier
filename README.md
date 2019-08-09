# identifier

Identify your media regardless of the metadata provider.

## What is this?

Identifier identifies media using a metadata provider such as; TVDB, Kitsu, IMDB, and etc, and turns it into
a standardized format that is then used by the Triton Media stack.

Identifier recieves jobs over AMQP to identify media then stores that inside PostgreSQL.

## License

Apache-2.0