
# v0.4 - 28 September 2016

* Add an "indir" block option to control the execution directory for a block.
* Fix some formatting issues in notifications (thanks @stereosteve)


# v0.3 - 8 April 2016

* Modd no longer exits when there are script errors on first run. Instead,
blocks with errors will be progressively started when there are applicable
changes.
* +onchange option to prep commands tells modd skip execution on startup, and
run only when there's a detected change. (thanks Thomas B Homburg
<thomas@homburg.dk>)
* @shell magic variable to switch the shell used to execute commands. Current
options are "bash" and "exec". (thanks Daniel Theophanes
<kardianos@gmail.com>)
* Modd now uses an exponential backoff strategy for daemon restarts (Josh
Bleecher Snyder <josharian@gmail.com>)
* Bugfix: Fix a format string issue that could cause some program output on the
console to be corrupted. (thanks Yoav Alon <yoava333@gmail.com>)


# v0.2 - 11 February 2016

* Windows support - thanks to @mattn for getting the ball rolling
* Fix a serious bug that prevented recursive watching on Linux
* Show full, variable expanded commands in logs
* Use slash-delimited paths throughout, even on Windows
* Properly handle CRLF line endings in config files
* Many, many small bugfixes and improvements
