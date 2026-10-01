# The filename field on the API is "/var/log/kafka/server.log" (leading slash).
# The import ID is cluster_type/cluster_name/filename, so the filename segment
# reproduces that leading slash as a double slash after cluster_name.
terraform import axonops_logcollector.server_log "kafka/my-kafka-cluster//var/log/kafka/server.log"
