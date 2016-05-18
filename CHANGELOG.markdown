# 0.5.20

* Realtime pipe fix. Revert to using Extrafiles for command pipe instead of Cmd.StdinPipe

# 0.5.19

* Increase statsd event limit from 500 to 1000, scout-client includes server_metrics 1.2.13

# 0.5.18

* Change logrotate for scout_streamer.log, upgrade to scout-client 6.2.1, server_metrics 1.2.12

# 0.5.17

* Include scout-client 6.2.0, fix scoutd version report

# 0.5.16

* Report loop bug fix to always fire on :00 second mark

# 0.5.15

* Bug fixes: pusher statsd delete metric, logrotate uses scoutctl, more scout-client troubleshoot info

# 0.5.14

Timer fixes in scoutd

# 0.5.13

* Ability to delete statsd metrics over pusher

# 0.5.12

* Version bump to add Ubuntu Vivid package

# 0.5.11

* Supports custom plugin testing through command line

# 0.5.10

* Updates server_metrics to 1.2.11, ignoring docker/lxc veth interfaces

# 0.5.9

* Metrics created from Timing events identify as type 'timer' in metrics payload

# 0.5.8

* Only report internal statsd metrics if events are being sent to statsd

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
