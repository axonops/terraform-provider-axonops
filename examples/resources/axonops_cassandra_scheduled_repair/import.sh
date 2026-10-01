# Preferred (v0.3+): cluster_type/cluster_name/tag
terraform import axonops_cassandra_scheduled_repair.monthly cassandra/my-cassandra-cluster/monthly-full-repair

# Legacy (pre-v0.3, still accepted): cluster_name/tag
terraform import axonops_cassandra_scheduled_repair.monthly my-cassandra-cluster/monthly-full-repair
