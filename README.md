[![Travis Build Status](https://travis-ci.org/cortesi/modd.svg?branch=master)](https://travis-ci.org/cortesi/modd)
[![Appveyor Build status](https://ci.appveyor.com/api/projects/status/1k7fk4j48oepvubo?svg=true)](https://ci.appveyor.com/project/cortesi/modd)


Modd is a developer tool that triggers commands and manages daemons in response to
filesystem changes.

If you use modd, you should also look at
[devd](https://github.com/cortesi/devd), a compact HTTP daemon for developers.
Devd integrates with modd, allowing you to trigger in-browser livereload with
modd.

The repo contains a set of example *modd.conf* files that you can look at for a
quick idea of what modd can do:

Example                                      | Description
-------------------------------------------- | -------
[frontend.conf](./examples/frontend.conf)    | A front-end project with React + Browserify + Babel. Modd and devd replace many functions of Gulp/Grunt.
[go.conf](./examples/go.conf)                | Live unit tests for Go.
[python.conf](./examples/python.conf)        | Python + Redis, with devd managing livereload.



# Install

Modd is a single binary with no external dependencies, released for OSX,
Windows, Linux, FreeBSD, NetBSD and OpenBSD. Go to the [releases
page](https://github.com/cortesi/modd/releases/latest), download the package
for your OS, and copy the binary to somewhere on your PATH.

If you have a working Go installation, you can also say

    $ go get github.com/cortesi/modd/cmd/modd

Note that by default modd uses either *bash* or *sh*. To use them they must be
on your PATH. To avoid using "bash" set `@shell = exec` in your "modd.conf"
file. On Windows, one easy way to install bash is to use
[Babun](https://babun.github.io/).


# Quick start

Put this in a file called *modd.conf*:

```
**/*.go {
    prep: go test @dirmods
}
```

Now run modd like so:

```
$ modd
```

The first time modd is run, it will run the tests of all Go modules. Whenever
any file with the .go extension is modified, the "go test" command will be run
only on the enclosing module.

To avoid shelling out to bash the following will execute "go test" directly.
```
@shell = exec
**/*.go {
    prep: go test @dirmods
}
```

# Leisurely start

On startup, modd looks for a file called *modd.conf* in the current directory.
This file has a simple but powerful syntax - one or more blocks of commands,
each of which can be triggered on changes to files matching a set of file
patterns. Commands have two flavors: **prep** commands that run and terminate
(e.g. compiling, running test suites or running linters), and **daemon**
commands that run and keep running (e.g databases or webservers). Daemons are
sent a SIGHUP (by default) when their block is triggered, and are restarted if
they ever exit.

Prep commands are run in order of occurrence. If any prep command exits with an
error, execution of the current block is stopped immediately. If all prep
commands succeed, any daemons in the block are restarted, also in order of
occurrence. If multiple blocks are triggered by the same set of changes, they
too run in order, from top to bottom.

Here's a modified version of the *modd.conf* file I use when hacking on devd.
It runs the test suite whenever a .go file changes, builds devd whenever a
non-test file is changed, and keeps a test instance running throughout.

```
**/*.go {
    prep: go test @dirmods
}

# Exclude all test files of the form *_test.go
**/*.go !**/*_test.go {
    prep: go install ./cmd/devd
    daemon +sigterm: devd -m ./tmp
}
```

The **@dirmods** variable expands to a properly escaped list of all directories
containing changed files. When modd is first run, this includes all directories
containing matching files. So, this means that modd will run all tests on
startup, and then subsequently run the tests only for the affected module
whenever there's a change. There's a corresponding **@mods** variable that contains all changed files.

Note the *+sigterm* flag to the daemon command. When devd receives a SIGHUP
(the default signal sent by modd), it triggers a browser livereload, rather
than exiting. This is what you want when devd is being used to serve a web
project you're hacking on, but when developing devd _itself_, we actually want
it to exit and restart to pick up changes. We therefore tell modd to send a
SIGTERM to the daemon instead, which causes devd to exit and be restarted by
modd.


# File watch patterns

Modd's change detection algorithm batches up changes until there is a lull in
filesystem activity - this means that coherent processes like compilation and
rendering that touch many files are likely to trigger commands only once.
Patterns therefore match on a batch of changed files - when the first match in
a batch is seen, the block is triggered.

Patterns and the paths they match against are always in slash-delimited form,
even on Windows.

### Quotes

File patterns can be naked or quoted strings. Quotes can be either single or
double quotes, and the corresponding quote mark can be escaped with a backslash
within the string:

```
"**/foo\"bar"
```

### Negation

Patterns can be negated with a leading **!**. For quoted patterns, the
exclamation mark goes outside of the quotes. So, this matches all files
recursively, bar those with a .html extension and those in the **docs**
directory.

```
** !**/*.html !"docs/**"
```

Negations are applied after all positive patterns - that is, modd collects all
files matching the positive patterns, then removes files matching the negation
patterns.

### Default ignore list

Common nuisance files like VCS directories, swap files, and so forth are
ignored by default. You can list the set of ignored patterns using the **-i**
flag to the modd command. The default ignore patterns can be disabled using the
special **+noignore** flag, like so:

```
.git/config +noignore {
    prep: echo "git config changed"
}
```

### Empty match pattern

If no match pattern is specified, prep commands run once only at startup, and
daemons are restarted if they exit, but won't ever be explicitly signaled to
restart by modd.

```
{
    prep: echo hello
}
```


### Syntax

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


# Blocks

Each file match pattern specification has an associated block, which is
enclosed in curly brackets. Blocks contain commands and block-scoped options.

Commands are shell scripts specified in-line in the *modd.conf* file. They are
executed in **bash** (or **sh** as a fallback) by default. Bash can be bypassed
and the commands executed directly by setting `@shell = exec`. If bash is used
it is assumed to be on the
user's path, and inherit the parent's environment. Single-line commands don't
need to be quoted:

```
prep: echo "i'm now rebuilding" | tee /tmp/output
```

Newlines can be escaped with a backslash for multi-line commands:

```
prep: ls \
        -l \
        -a
```

You can also enclose commands in single or double quotes, letting easily
specify compound, multi-statement commands. These can contain anything you'd
normally put in a shell script, and the same quoting and escaping conventions apply.

```
prep: "
    ls \
        -l \
        -a
    echo \"hello again\"
    echo \"hello yet again\"
"
```


### Prep commands

All prep commands in a block are run in order before any daemons are restarted.
If any prep command exits with an error, execution stops.

There following variables are automatically generated for prep commands

Variable      | Meaning
------------- | -------
@mods         | On first run, all files matching the block patterns. On subsequent change, a list of all modified files.
@dirmods      | On first run, all directories containing files matching the block patterns. On subsequent change, a list of all directories containing modified files.

All file names in variables are relative to the current directory, and
shell-escaped for safety. All paths are in slash-delimited form on all
platforms.

Given a config file like this, modd will run *eslint* on all .js files when
started, and then after that only run *eslint* on files if they change:

```
**/*.js {
    prep: eslint @mods
}
```

By default, prep commands are executed on the initial run of modd. The
`+onchange` option can be used to skip the initial run, and only execute when
there is a detected change.

```
*.go {
	# only trigger on file changes
	prep +onchange: go test
}
```


### Daemon commands

Daemons are executed on startup, and are restarted by modd whenever they exit.
When a block containing a daemon command is triggered, modd sends a signal to
the daemon process. If the signal causes the daemon to exit, it is immediately
restarted by modd - however, it's also common for daemons to do other useful
things like reloading configuration in response to signals.

The default signal used is SIGHUP, but the signal can be controlled using
modifier flags, like so:

```
daemon +sigterm: mydaemon --config ./foo.conf
```

The following signals are supported: **sighup**, **sigterm**, **sigint**,
**sigkill**, **sigquit**, **sigusr1**, **sigusr2**, **sigwinch**.


### Controlling log headers

Modd outputs a short header on the terminal to show which command is
responsible for output. This header is calculated from the first non-whitespace
line of the command - backslash escapes are removed from the end of the line,
comment characters are removed from the beginning, and whitespace is stripped.
Using the fact that the shell itself permits comments, you can completely
control the log display name.

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

### Options

The only block option at the moment is **indir**, which controls the execution
directory of a block. Modd will change to this directory before executing
commands and daemons, and change back to the previous directory afterwards.

The directory specification follows the same conventions as commands, and can
be enclosed in quotes to span multiple lines.

```
{
    indir: ./my/directory
    prep: ls
}
```


# Variables

You can declare variables like this:

```
@variable = value
```

Variables can only be declared in the global scope (i.e. not inside blocks).
All values are strings and follow the same semantics as commands - that is,
they can have escaped line endings, or be quoted strings. Variables are read
once at startup, and it is an error to re-declare a variable that already
exists.

You can use variables in commands like so:

```
@dst = ./build/dst
** {
    prep: ls @dst
}
```

There is a special "@shell" variable that when set determines how commands are
executed. Valid values are:
```
# Execute commands directly from go.
@shell = exec

# Pass commands on to bash or sh for execution.
@shell = bash
```

# Desktop Notifications

When the **-n** flag is specified, modd sends anything sent to *stderr* from
any prep command that exits abnormally to a desktop notifier. Since modd
commands are shell scripts, you can redirect or manipulate output to entirely
customize what gets sent to notifiers as needed.

At the moment, we support [Growl](http://growl.info/) on OSX, [osascript display notification](https://developer.apple.com/library/content/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/DisplayNotifications.html) on OSX and
[libnotify](https://launchpad.net/ubuntu/+source/libnotify) on Linux and other
Unix systems.

## Growl

For Growl to work, you will need Growl itself to be running, and have the
**growlnotify** command installed. Growlnotify is an additional tool that you
can download from the official [Growl
website](http://growl.info/downloads.php).


## Libnotify

Libnotify is a general notification framework available on most Unix-like
systems. Modd uses the **notify-send** command to send notifications using
libnotify. You'll need to use your system package manager to install
**libnotify**.


# Development

The scripts used to build this package for distribution can be found
[here](https://github.com/cortesi/godist). External packages are vendored using
[gvt](https://github.com/FiloSottile/gvt), and I use
[deplist](https://github.com/cespare/deplist) to make sure all dependencies are
covered.
