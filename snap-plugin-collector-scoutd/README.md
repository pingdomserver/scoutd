# Scout Snap collector plugin
Scout go daemon transformed into snap's collector plugin. Initial version just reads the scoutd's
configuration file and fires the ruby plugin.

## Prerequisites
Dockerfile configuration provides the snap and go tooling (for compiling the snap
plugin inside of the container). It downloads binary distribution of go from the official website
and installs it inside of the folder `/usr/local/`.

### Step-by-step docker initialization
Build and start the docker image using provided Dockerfile:
1. `./build.sh`
2. `docker run -it -v psm_dev/code:/opt/workspace solarwinds/snap_scout:xenial /bin/bash --login` #
   opens a shell session inside of the container
   
Rest of the commands in this document are invoked inside of the docker container.

### Scoutd initialization
Before using the snap plugin we need to install scoutd inside of the container.

`./scout_install.sh <psm_key>`

## Starting the plugin
The file task-snap-scout.yaml contains task's configuration for snap.
Starting the plugin:
1. `snapteld -l 1 -t 0 &` # enable snap
2. `snaptel plugin load <compiled_go_code, e.g. main>`
  * optional: list plugins `snaptel plugin list`, list plugin metrics `snaptel metric list`
3. `snaptel task create -t task-snap-scout.yaml`
  * optional: list tasks `snaptel task list`
4. `snaptel taks list` # check if task is in running state
4. `tail -f /var/log/snap/snapteld.log` # see what snap is doing

After these steps, the scout collector plugin should be operational and it should be collecting data
every second (modify task-snap-scout.yaml file for different interval).
Collector uses scoutd's log file: `tail -f /var/log/scout/scoutd.log`
