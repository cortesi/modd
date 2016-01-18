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

When modd is started, it looks for a file called **modd.conf** in the current
directory. This file has a simple but powerful syntax - one or more blocks
specifying commands to run whenever files matching a set of patterns change.

Here's a simple example:

    **/*.go {
        prep: go test
    }

This config file looks for all changes to .go files recursively. When a change
is detected, a **prep** command runs the unit test suite with *go test*.  A
**prep** command is simply a command that's expected to terminate - for
example, compiling, running a test suite or running a linter.

We can now start modd, like so:

![screenshot](doc/modd-example1.png "modd in action")

Modd runs the command once on startup, and then waits for modifications that
match the specified patterns before running again.

There's a second command type - **daemon** - for commands that you want to keep
running. Daemons are sent a SIGHUP (by defaut) when triggered, and will also be
restarted whenever they exit. Below is a simplified version of the modd.conf
file I use when hacking on devd. It runs the test suite, builds and installs
devd, and then runs an instance for testing:

    **/*.go {
        prep: go test
        prep: go install ./cmd/devd
        daemon: devd -m ./tmp
    }

Output looks like this:

![screenshot](doc/modd-example2.png "modd in action")

All prep commands in a block are run in order of occurrence before any daemon
is restarted. If any prep command exits with an error, execution is stopped.

There's one small problem with the devd example - when devd sees a SIGHUP it
doesn't exit, it triggers browser livereload. This is precisely what you want
when devd is being used to serve a web project you're hacking on, but for devd
*development*, we actually want it to exit. So, we tell modd to send a SIGTERM
to the daemon instead, which has the desired result:

    **/*.go {
        prep: go test
        prep: go install ./cmd/devd
        daemon +sigterm: devd -m ./tmp
    }


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

The modd.conf file format is very simple.






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
