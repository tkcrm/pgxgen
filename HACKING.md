# Development

To hack on `pgxgen` you will need a postgres database to test against.
Make sure you have postgres installed, then from the root of the repo run

```bash
> createdb pgxgen_development
> psql pgxgen_development < cmd/pgxgen/test/db.sql
```

You will need to re-run the command loading `cmd/pgxgen/test/db.sql` any time
you make a change to the test data or schema defined in that file. Once
the test database is set up you can run all tests by doing

```bash
> go generate ./...
> go test ./...
```

Most of the tests are defined in the `cmd/pgxgen/test` package, so if you
want to focus on a specific test, you will probably want to look there.

## With Docker

If you want, you can use almost the same docker set up that `pgxgen` uses for
continuous integration for local development. The setup for local development
differs in that it uses a docker volume so that changes on your local file system
are picked up and tested without needing to re-build the image each time.

First, build the development docker image:

```bash
> docker-compose build
```

once you have the image built, you can either run the tests
start to finish with

```bash
> docker-compose run test11
```

or debug interactively by opening a bash shell with

```bash
> docker-compose run test11 bash
```

The full test suite runs against multiple versions of postgres. The above examples
just work with postgres 11, but you can see the other versions of postgres that
`pgxgen` is tested against by reading `docker-compose.yml`.

## CLI Tests

`pgxgen/test/cli_test.go` contains some end-to-end tests of the `pgxgen` command
line utility. These tests spawn subprocesses and then perform some assertions
about the stdout, stderr, and return code of the subprocesses. If one of them
fails, it will print the command used to execute the failing test, but you won't
be able to re-run the command because the test will have cleaned up after
itself. You can enable re-running the cli tests with

```bash
> cd cmd/pgxgen/test
> pgxgen_DEBUG_CLI=1 go test --run TestCLI
```

This will both prevent the cli tests from removing their scratch dir and
compile the binary under test in a debugger friendly format so that you can
immediately point your delve at it.

## Example Tests

A great way to both document pgxgen and provide more end-to-end test coverage is to add a
new example to the `examples` directory. Examples must follow a very specific format in order
to be run by the test suite. This format is specified in the module comment at the top of
`examples/examples_test.go`. If you are working on developing just a specific example, it is
worth knowing about the `pgxgen_TEST_EXAMPLE` environment variable, which can be used to focus
on one or more examples rather than running the whole suite every time.

## Errors in the Generated Code

When modifying the output of the codegenerator, you are likely to introduce compile
errors in the generated code. Because we run the `go/format` package over our output
before landing it to a disk you won't be able to debug the issue by looking at the
generated file by default. In order to make this easier, you can set `pgxgen_GOFMT=off`
in the environment. This will prevent pgxgen from formatting the generated code
and make it easier to debug the output from pgxgen.

## A word about CI

`pgxgen` runs a number of different CI checks in some docker containers orchestrated by
`.circleci/docker-compose.yml`. Using docker adds some inefficiency and slows down the
CI checks a bit, but hopefully this is made up for by the fact that the `docker-compose.yml`
file in the repo root makes debugging CI jobs easier. We maintain two different compose files
so that we can share the source between the host and the container during local development.

## Testing Non-Default Drivers

`pgxgen` supports using both `github.com/lib/pq` and `github.com/jackc/pgx/v4/stdlib` as
database drivers. `jackc/pgx` is recommended because `lib/pq` is unmaintained.

The example tests all use the recommended driver (`jackc/pgx`) for testing to keep the example code
simple. The main test suite `cmd/pgxgen/test` is parameterized over the driver though. It
allows you to you set the driver name via the `DB_DRIVER` environment variable. You can
either set this variable to `postgres` (to use `lib/pq`) or `pgx` (to use
`github.com/jackc/pgx/v4/stdlib`).

The `tools/test.bash` script runs the test suite both ways.

# Philosophy

Certain aspects of the style in which `pgxgen` is developed are intentionally divergent
from common practices in other places. A few of these are called out here with some
explanation as to why things are the way they are.

## Testing

`pgxgen` uses the built in go unit test framework. This is good because tests are code, not English,
and trying to make them look like English tens to encourage tests that fail to explore the state
space programatically. `pgxgen` will never use a test framework like ginkgo. A flexible assertions
framework like gomega may be considered, but it is likely better to leverage libraries like
`google/go-cmp` and pattern matching DSLs like regular expressions. Using a randomized testing
framework in the spirit of quickcheck or a fuzz tester would definitely be considered.

pgxgen has no external components that cannot be easily controlled, so `pgxgen` tests do not
and will never use mocks. Mocking should be a last resort when testing with the real thing is
impractical. This avoidance of mocks has the great side benefit of preventing the codebase
from sprouting superfluous interfaces.
