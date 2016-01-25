[![Build Status](https://drone.io/github.com/cortesi/modd/status.png)](https://drone.io/github.com/cortesi/modd/latest)

Modd is a developer tool that runs commands and manages daemons in response to
filesystem changes.

If you use modd, you should also look at
[devd](https://github.com/cortesi/devd), a compact HTTP daemon for developers.
Devd integrates with modd, allowing you to trigger in-browser livereload with
modd.

**Modd has not been released yet - the design is stabilising and I should have
version 0.1 out the door soon.**


# Install

Modd is a single binary with no external dependencies, released for OSX and
Linux. Go to the [releases
page](https://github.com/cortesi/modd/releases/latest), download the package
for your OS, and copy the binary to somewhere on your PATH.

If you have a working Go installation, you can also say

    go get github.com/cortesi/modd/cmd/modd


# Quick start

Put this in a file called *modd.conf*:

```
**/*.go {
    prep: go test ./...
}
```

Now run modd like so:

![screenshot](doc/modd-example1.png "modd in action")

Whenever any file with the .go extension is modified, the "go test" command
will be run.


# Leisurely start

When modd is started, it looks for a file called *modd.conf* in the current
directory. This file has a simple but powerful syntax - one or more blocks of
commands, each of which can be triggered on changes to files matching a set of
file patterns. Commands have two flavors: **prep** commands that run and
terminate (e.g. compiling, running test suites or running linters), and
**daemon** commands that run and keep running (e.g databases or webservers).
Daemons are sent a SIGHUP (by default) when their block is triggered, and will
be restarted if they ever exit.

Prep commands are run in order of occurrence. If any prep command exits with an
error, execution of the current block is stopped immediately. If all prep
commands succeed, any daemons in the block are restarted, also in order of
occurrence. If multiple blocks are triggered by the same set of changes, they
too run in order, from top to bottom.

Let's look at a simplified version of the *modd.conf* file I use when hacking
on devd. It runs the test suite, builds and installs devd, and keeps a test
instance running throughout:

```
**/*.go {
    prep: go test
    prep: go install ./cmd/devd
    daemon: devd -m ./tmp
}
```

This works, but there's one small problem - when devd gets a SIGHUP (the
default signal sent by modd), it doesn't exit, it triggers browser livereload.
This is precisely what you want when devd is being used to serve a web project
you're hacking on, and is the reason why devd is so useful in conjunction with
modd. However, when developing devd _itself_, we actually want it to exit and
restart to pick up changes. So, we tell modd to send a SIGTERM to the daemon
instead, which has the desired result:

```
**/*.go {
    prep: go test
    prep: go install ./cmd/devd
    daemon +sigterm: devd -m ./tmp
}
```

Next, it's not really necessary to do an install and restart the daemon if
we've only changed a unit test file. Let's change the config file so modd runs
the test suite whenever we change any source file, but skips the rest if we've
only modified a test specification. We do this by excluding test files with the **!** operator, and adding another block.

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

Lastly, let's say we want to run *gofmt* to auto-format files whenever they are
modified. We can use the **|MODD|** marker in prep commands for this. On first
run it is expanded to a shell-safe list of all matching files on disk.
Subsequently, when responding to an actual change, it expands to a list of
files that have been modified or added. Our final *modd.conf* file looks like
this:

```
**/*.go {
    prep: gofmt -w |MODD|
    prep: go test
}

# All test files are of the form *_test.go
**/*.go !**/*_test.go {
    prep: go install ./cmd/devd
    daemon +sigterm: devd -m ./tmp
}
```


# Config file format

A modd config file consists of blocks that start with zero or more file watch
patterns, and each contain a set of **prep** and **daemon** commands.


## File watch patterns

Modd's change detection algorithm batches up changes until there is a lull in
filesystem activity - this means that coherent processes like compilation and
rendering that touch many files are likely to trigger commands only once.
Patterns therefore match on a batch of changed files - when the first match in
a batch is seen, the block is triggered.

```
# File patterns can be naked or quoted
**/*.js "**/*.html" {
    prep: echo hello
}

# Recursively matches any file
** {
    prep: echo hello
}

# No match pattern. Prep commands run once only at startup. Daemons are
# restarted if they exit, but won't ever be explicitly signaled to restart by
# modd.
{
    prep: echo hello
}
```

Patterns can be negated with a leading **!**. For quoted patterns, the
exclamation mark goes outside of the quotes:

```
** !**/*.html !"docs/**" {
    prep: echo changed
}
```

Negations are applied after all positive patterns - that is, modd collects all
files matching all the positive patterns, regardless of order, then remove
files matching the negation patterns.

Common nuisance files like VCS directories, swap files, and so forth are
ignored by default. You can list the set of ignored patterns using the **-i**
flag to the modd command. The default ignore patterns can be disabled using the
special **+noignore** flag, like so:

```
.git/config +noignore {
    prep: echo "git config changed"
}
```

File patterns support the following syntax:

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


## Commands

Commands are shell scripts specified in-line in the *modd.conf* file. They are
executed in **bash**, which is assumed to be on the user's path, and inherit
the parent's environment. Single-line commands don't need to be quoted:

```
prep: echo "i'm now rebuilding" | tee /tmp/output
```

Multi-line commands must be quoted using single or double quotes. Within a
multi-line command, the enclosing quote type can be backslash-escaped.

```
prep: "
    ls \
        -l \
        -a
    echo \"hello again\"
"
```


### prep

Within each block, all prep commands are run in order of occurance before any
daemons are restarted. If any prep command exits with an error, execution
stops.

Prep commands can include a special marker **|MODD|**, which is replaced with a
shell-escaped list of files that have changed or been added since the last run.
When modd is first run, and the prep command is not being triggered by a
change, the marker is replaced by all matching files on disk. So, given a
config file like this, modd will run eslint on all .js files when started, and
then after that only run eslint on files if they change:

```
**/*.js {
    prep: eslint |MODD|
}
```

### daemon

Daemons are executed on startup, and are restarted by modd if they exit.
Whenever a block containing a daemon is triggered, modd sends a signal to the
daemon process. It's up to the daemon how the signal is handled - for example,
a SIGHUP might cause a daemon to reload config without restarting, or it could
simply exit, in which case modd will restart it automatically.

By default, modd sends a SIGHUP, but the signal can be controlled using
modifier flags, like so:

```
daemon +sigterm: mydaemon --config ./foo.conf
```

The following signals are supported: **sighup**, **sigterm**, **sigint**,
**sigkill**, **sigquit**, **sigusr1**, **sigusr2**, **sigwinch**.


### Log headers

On the terminal, modd outputs a short header to show which command is
responsible for output. This header is calculated from the input command using
the first significant non-whitespace line of text - backslash escapes are
removed from the end of the line, comment characters are removed from the
beginning, and whitespace is stripped. Using the fact that the shell itself
permits comments, you can completely control the log display name.

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
