[![Build Status](https://drone.io/github.com/cortesi/modd/status.png)](https://drone.io/github.com/cortesi/modd/latest)

# modd

Modd triggers commands and manages daemons in response to filesystem changes.

# Install

Go to the [releases page](https://github.com/cortesi/modd/releases/latest),
download the package for your OS, and copy the binary to somewhere on your
PATH.

If you have a working Go installation, you can also say

    go get github.com/cortesi/modd/cmd/modd


# Quick start

Modd is driven by a **modd.conf** file with a simple but powerful syntax. Let's look at some examples.

This config file looks for all changes to .go files recursively in all
directories. When a change is detected, a **prep** command runs the unit test
suite with *go test*. A **prep** command is simply a command that's expected to
terminate, as opposed to a **daemon** which modd expects to keep running.

```
**/*.go {
    prep: go test ./...
}
```

Say the project we're working on is a daemon, and we'd like to keep a test instance of it running. The config file might look like this:

```
**/*.go {
    prep: go test ./...
    prep: go install ./cmd/myserver
    daemon: myserver /static/data
}
```

All **prep** commands in a block are run in order of occurrence before any
**daemon** is restarted. If any prep command exits with an error, execution is
stopped.

All processes inherit the parent environment.


# Features

### Cross-platform and self-contained

Modd is a single statically compiled binary with no external dependencies, and
is released for OSX, Linux and Windows.


### Works well with devd

Modd's sister project is [devd](https://github.com/cortesi/devd), a compact HTTP daemon for developers. Devd integrates with modd, allowing you to trigger in-browser livereload after static resource rebuilds complete.

### Designed for the terminal

This means no daemonization, and output that is designed to be read in the
terminal by a developer.


### Does file change detection right

Or at least tries. Usefully responding to file system changes is a hairy,
knotty, horrible problem, and most tools similar to modd simply don't get it
right. Modd aims to do the best possible job across all platforms for typical
developer work patterns. It ignores temporary files, VCS directories, swap
files and many other nuisances by default. Its detection algorithm waits for a
lull in filesystem activity - so that events are triggered **after** render or
compilation processes that may touch many files.


### Restart daemons tenderly and correctly

You choose how daemons should be restarted. By default, daemons are sent a
SIGHUP so they have the opportunity to reload config without a restart, and he
actual signal sent is configurable per-daemon.



# Config file format




## File watch patterns

Watch patterns support the following terms:

Term          | Meaning
------------- | -------
`*`           | any sequence of non-path-separators
`**`          | any sequence of characters, including path separators
`?`           | any single non-path-separator character
`[class]`     | any single non-path-separator character against a class of characters
`{alt1,...}`  | any of the comma-separated alternatives - to avoid conflict with the block specification, patterns with curly-braces should be enclosed in quotes

Any character with a special meaning can be escaped with a backslash (`\`). Character classes support the following:

Class      | Meaning
---------- | -------
`[abc]`    | any character within the set
`[a-z]`    | any character in the range
`[^class]` | any character which does *not* match the class
