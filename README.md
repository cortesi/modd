[![Build Status](https://drone.io/github.com/cortesi/modd/status.png)](https://drone.io/github.com/cortesi/modd/latest)

# modd: trigger commands when files change

# Install

Go to the [releases page](https://github.com/cortesi/modd/releases/latest),
download the package for your OS, and copy the binary to somewhere on your
PATH.

If you have a working Go installation, you can also say

    go get github.com/cortesi/modd/cmd/modd


# Quick start



# Features


### Cross-platform and self-contained

Modd is a single statically compiled binary with no external dependencies, and
is released for OSX, Linux and Windows.


### Designed for the terminal

This means no config file, no daemonization, and logs that are designed to be
read in the terminal by a developer.


### Does file change detection right



### Restart daemons tenderly and correctly



### Separate daemons and prep commands



## Excluding files

The **-x** flag supports the following terms:

Term          | Meaning
------------- | -------
`*`           | matches any sequence of non-path-separators
`**`          | matches any sequence of characters, including path separators
`?`           | matches any single non-path-separator character
`[class]`     | matches any single non-path-separator character against a class of characters
`{alt1,...}`  | matches a sequence of characters if one of the comma-separated alternatives matches

Any character with a special meaning can be escaped with a backslash (`\`). Character classes support the following:

Class      | Meaning
---------- | -------
`[abc]`    | matches any single character within the set
`[a-z]`    | matches any single character in the range
`[^class]` | matches any single character which does *not* match the class
