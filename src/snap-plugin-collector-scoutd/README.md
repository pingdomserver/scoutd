Scout Snap collector plugin
---------------------------
Scout go daemon transformed into snap's collector plugin. Initial version just reads the scoutd's
configuration file and fires the ruby plugin.

## Configuration
Use the provided Docker image for snap and scoutd's prerequisites.
The file task-snap-scout.yaml contains task's configuration for snap.
Step to enable the plugin:

1. Enable snap: `snapteld -l 1 -t 0`
2. Load scout collector plugin: `snaptel plugin load`
  * optional: list plugins `snaptel plugin list`, list plugin's metrics `snaptel metric list`
3. Create snap's task: `snaptel task create -t task-snap-scout.yaml`
  * optional: list tasks `snaptel task list`

After these steps the scout collector plugin should be operational and it should be collecting data
in 1s interval (modify task-snap-scout.yaml file for different interval).
Collector uses scoutd's log file: `tail -f /var/log/scout/scoutd.log`
