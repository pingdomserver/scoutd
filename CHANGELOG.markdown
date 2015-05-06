# 0.5.7

* Disable reporting of all but two internal statsd metrics

# 0.5.6

* Version bump for scout-package changes

# 0.5.5

* Built-in statsd server enabled by default
* CLI options for statsd, augo config generator understands statsd settings

# 0.5.4

* Enforce event limit in statsd server. Default: 500
* Snap report loops to :00 of minute
* Calculate 95 percentile of statsd timers
* Track and report internal statsd metrics

# 0.5.3

* Initial statsd server implementation

# 0.4.19

* Version bump for scout-package changes only

# 0.4.18

* scoutd can run under scoutd_supervise for sysvinit systems

# 0.4.17

* Fix bug in Realtime timeout exception handling

# 0.4.16

* Add LogLevel option to invoke scout-client with debug logging
* Log agent output on unsuccessful checkins even on normal LogLevel
* Log agent output when launching realtime process fails

# 0.4.15

* update the server_metrics code to find processes when run inside a Docker container
