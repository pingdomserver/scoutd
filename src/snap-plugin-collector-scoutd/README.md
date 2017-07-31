# Scout Snap collector plugin
Scout go daemon transformed into snap's collector plugin. Initial version just reads the scoutd's
configuration file and fires the ruby plugin.

## Prerequisites
Dockerfile configuration provides the `snap` and `go` tooling (for compiling the snap
plugin inside of the container). It downloads binary distribution of `go` from the official website
and installs it inside of the folder `/usr/local/`.

### Step-by-step docker initialization
Build and start the docker image using provided Dockerfile:
1. `cd docker`
2. `./build.sh`
3. `docker run -it -v psm_dev/code:/opt/workspace solarwinds/snap_scout:xenial /bin/bash --login` #
   opens a shell session inside of the container

## Configuration
Use the provided Docker image for snap and scoutd's prerequisites.

The file task-snap-scout.yaml contains task's configuration for snap.
Step to enable the plugin:

1. `snapteld -l 1 -t 0` # enable snap
2. `snaptel plugin load <compiled_go_code, e.g. main>` # load scout collector plugin
  * optional: list plugins `snaptel plugin list`, list plugin metrics `snaptel metric list`
3. `snaptel task create -t task-snap-scout.yaml` # create snap's task
  * optional: list tasks `snaptel task list`
4. `snaptel taks list` # check if task is in running state
4. `tail -f /var/log/snap/snapteld.log` # see what snap is doing

After these steps, the scout collector plugin should be operational and it should be collecting data
in 1s interval (modify task-snap-scout.yaml file for different interval).
Collector uses scoutd's log file: `tail -f /var/log/scout/scoutd.log`
