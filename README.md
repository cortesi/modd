[![Build Status](https://drone.io/github.com/cortesi/modd/status.png)](https://drone.io/github.com/cortesi/modd/latest)

# modd

A dev tool that runs commands and manages daemons in response to filesystem changes.

# Install

Modd is a single statically compiled binary with no external dependencies, and
is released for OSX and Linux. Go to the [releases
page](https://github.com/cortesi/modd/releases/latest), download the package
for your OS, and copy the binary to somewhere on your PATH.

If you have a working Go installation, you can also say

    go get github.com/cortesi/modd/cmd/modd


# Quick start

Put this in a file called **modd.conf**:

```
**/*.go {
    prep: go test
}
```

Run modd like so:

![screenshot](doc/modd-example1.png "modd in action")

Whenever any file with the .go extension is modified, the "go test" command
will be run.


# Leisurely start

When modd is started, it looks for a file called **modd.conf** in the current
directory. This file has a simple, powerful syntax - one or more blocks of
commands, each of which can be triggered on changes to files matching a set of
file patterns. Commands have two flavors: **prep** commands that run and
terminate (e.g. compiling, running test suites or running linters), and
**daemon** commands that run and keep running (e.g databases or webservers).
Daemons are sent a SIGHUP (by default) when their block is triggered, and will
be restarted if they exit unexpectedly.

All prep commands in a block are run in order of occurrence. If all prep
commands succeed, daemons are then restarted, also in order of occurrence. If
any prep command exits with an error, execution is stopped immediately. If
multiple blocks are triggered, they too run in order from top to bottom.


## An example

Let's start with a simplified version of the modd.conf file I use when hacking
on devd. It runs the test suite, builds and installs devd, and keeps a test
daemon instance running throughout:

```
**/*.go {
    prep: go test
    prep: go install ./cmd/devd
    daemon: devd -m ./tmp
}
```

This works, but there's one small problem - when devd gets a SIGHUP it doesn't
exit, it triggers browser livereload. This is *precisely* what you want when
devd is being used to serve a web project you're hacking on, and is the key
reason why devd is so useful in conjunction with modd. However, when developing
devd _itself_, we actually want it to exit and restart to pick up changes. So,
we tell modd to send a SIGTERM to the daemon instead, which has the desired
result:

```
**/*.go {
    prep: go test
    prep: go install ./cmd/devd
    daemon +sigterm: devd -m ./tmp
}
```

Now, it's not really necessary to do an install and restart the daemon if we've
only changed a unit test file. Let's change the config file so modd runs the
test suite whenever we change any source file, but skips the rest if we've only
modified a test specification. We do this by excluding test files with the **!** operator, and adding another block.

```
**/*.go {
    prep: go test
}

# All test files are of the form *_test.go
**/*.go !**/*_test.go {
    prep: go install ./cmd/devd
    daemon +sigterm: devd -m ./tmp
}
```


# Features


### Works well with programmers

Modd is designed to be a simple, reliable that does what's needed and gets out
of your way.


### Works well with devd

Modd's sister project is [devd](https://github.com/cortesi/devd), a compact
HTTP daemon for developers. Devd integrates with modd, allowing you to trigger
in-browser livereload after static resource rebuilds complete.

### Does file change detection right

Or at least tries. Usefully responding to file system changes is a hairy,
knotty, horrible problem, and most tools similar to modd simply don't get it
right. Modd aims to do the best possible job across all platforms for typical
developer work patterns. It ignores temporary files, VCS directories, swap
files and many other nuisances by default. Its detection algorithm waits for a
lull in filesystem activity so that events are triggered **after** render or
compilation processes that may touch many files. Modd tries to do the right
thing in corner cases, like receiving file modification notice while previously
triggered commands are being run.


# Config file format

A modd config file consists of one or more blocks, each starting with a set of
file watch patterns, and specifying a set of **prep** and **daemon** commands
to run. Here's an example showing all the basic features of the format:

```
# File patterns can be naked or quoted
**/*.js "**/*.html" {
    # Commands are executed in a shell, and can make full use of shell
    # capabilities like piping and output redirection
    prep: echo "i'm now rebuilding" | tee /tmp/output

    # Commands can be quoted, and can then span multiple lines. Commands are
    # shell scripts, executed in bash
    prep: "
        ls \
            -l \
            -a
        echo 'and hello again'
    "
}

# A double-asterisk recursively matches all files so this block will trigger on
# any change
** {
    prep: go test
}

# This is a special block with no file match pattern. This means prep commands
# run once only at startup, and daemons are kept running but never restarted by
# modd.
{
    prep: echo "i run exactly once"
}
```

## File watch patterns

Watch patterns support the following terms:

Term          | Meaning
------------- | -------
`*`           | any sequence of non-path-separators
`**`          | any sequence of characters, including path separators
`?`           | any single non-path-separator character
`[class]`     | any single non-path-separator character against a class of characters
`{alt1,...}`  | any of the comma-separated alternatives - to avoid conflict with the block specification, patterns with curly-braces should be enclosed in quotes

Any character with a special meaning can be escaped with a backslash (`\`).
Character classes support the following:

Class      | Meaning
---------- | -------
`[abc]`    | any character within the set
`[a-z]`    | any character in the range
`[^class]` | any character which does *not* match the class

Patterns can be negated with a leading **!**. For quoted patterns, the
exclamation mark goes outside of the quotes:

```
** !**/*.html !"My Documents/**" {
    prep: echo changed
}
```

Negations are applied after all positive patterns - that is, modd collects all
files matching all the positive patterns, regardless of order, then remove
files matching the negation patterns.


## Some notes on running commands

All commands are executed in **bash**, which is assumed to be somewhere on the
user's path. Fixing on one shell ensures that *modd.conf* files remain
portable, and can be used reliably regardless of the user's personal shell
choice. Processes inherit the parent's environment, so you can pass environment
variables down to commands like so:

```
env MYCONFIG=foo modd
```

On the terminal, modd outputs a short header to show which command is
responsible for a given line of output. This header is calculated from the
input command, using the first significant non-whitespace line of text -
backslash escapes are removed from the end of the line, comment characters are
removed from the beginning, and whitespace is stripped. Using the fact that the
shell itself permits comments, you can completely control the log display name.

```
{
    # This will show as "prep: mycommand"
    prep: "
        mycommand \
            --longoption 1 \
            --longoption 2
    "
    # This will show as "prep: daemon 1"
    prep: "
        # daemon 1
        mycommand \
            --longoption 1 \
            --longoption 2
    "
}
```
