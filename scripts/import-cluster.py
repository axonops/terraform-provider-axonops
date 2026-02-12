#!/usr/bin/env python3
"""
AxonOps Kafka Terraform Import Script

This script queries an AxonOps instance and generates Terraform configuration
files along with import commands for all resources in a Kafka cluster.

Usage:
    python import-cluster.py <axonops_host> <org_id> <cluster_name> <api_key> [output_dir]

Example:
    python import-cluster.py axonops.example.com myorg mycluster abc123 ./imported
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from typing import Any


def sanitize_name(name: str) -> str:
    """Convert a name to a valid Terraform resource name."""
    return name.replace('.', '_').replace('-', '_').replace(' ', '_')


def escape_hcl_string(value: str) -> str:
    """Escape a string for use in HCL (Terraform) configuration."""
    # Escape backslashes first, then double quotes
    return value.replace('\\', '\\\\').replace('"', '\\"')


def make_request(url: str, api_key: str) -> Any:
    """Make an authenticated API request."""
    req = urllib.request.Request(url)
    req.add_header('Authorization', f'AxonApi {api_key}')

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        print(f"  HTTP Error {e.code}: {e.reason} for {url}")
        return None
    except urllib.error.URLError as e:
        print(f"  URL Error: {e.reason} for {url}")
        return None
    except json.JSONDecodeError:
        return None


def generate_provider_tf(output_dir: str, host: str, protocol: str, org_id: str) -> None:
    """Generate provider.tf configuration."""
    content = f'''terraform {{
  required_providers {{
    axonops = {{
      source  = "hashicorp/axonops"
      version = "1.0.0"
    }}
  }}
}}

provider "axonops" {{
  api_key          = var.axonops_api_key
  axonops_host     = "{host}"
  axonops_protocol = "{protocol}"
  org_id           = "{org_id}"
}}

variable "axonops_api_key" {{
  description = "AxonOps API key"
  type        = string
  sensitive   = true
}}
'''
    with open(os.path.join(output_dir, 'provider.tf'), 'w') as f:
        f.write(content)


def import_topics(api_base: str, api_key: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import topics and return import commands."""
    print("Fetching topics...")
    url = f"{api_base}/kafka/{cluster_name}/topics"
    topics = make_request(url, api_key)

    if not topics or not isinstance(topics, list):
        print("  No topics found")
        return []

    resources = []
    imports = []

    for topic in topics:
        name = topic.get('name', '')

        # Skip internal topics
        if name.startswith('_') or name == '__consumer_offsets':
            continue

        safe_name = sanitize_name(name)
        partitions = topic.get('partitionCount', 1)
        replication = topic.get('replicationFactor', 1)

        # Fetch topic configs
        config_url = f"{api_base}/kafka/{cluster_name}/topics/{name}/configs"
        config_data = make_request(config_url, api_key)

        config_lines = []
        if config_data:
            topic_desc = config_data.get('topicDescription', [])
            if topic_desc and len(topic_desc) > 0:
                for entry in topic_desc[0].get('configEntries', []):
                    # Only include explicitly set configs
                    if entry.get('isExplicitlySet', False):
                        # Convert dots to underscores for Terraform
                        key = entry.get('name', '').replace('.', '_')
                        value = escape_hcl_string(entry.get('value', ''))
                        config_lines.append(f'    {key} = "{value}"')

        resource = f'''resource "axonops_kafka_topic" "{safe_name}" {{
  name               = "{name}"
  partitions         = {partitions}
  replication_factor = {replication}
  cluster_name       = "{cluster_name}"'''

        if config_lines:
            resource += '\n  config = {\n'
            resource += '\n'.join(config_lines)
            resource += '\n  }'

        resource += '\n}\n'

        resources.append(resource)
        imports.append(f'terraform import axonops_kafka_topic.{safe_name} "{cluster_name}/{name}"')

    if resources:
        with open(os.path.join(output_dir, 'topics.tf'), 'w') as f:
            f.write('# Kafka Topics\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} topics")

    return imports


def import_acls(api_base: str, api_key: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import ACLs and return import commands."""
    print("Fetching ACLs...")
    url = f"{api_base}/kafka/{cluster_name}/acls"
    data = make_request(url, api_key)

    if not data:
        print("  No ACLs found")
        return []

    resources_list = data.get('aclResources', [])
    if not resources_list:
        print("  No ACLs found")
        return []

    resources = []
    imports = []
    counter = 0

    for res in resources_list:
        res_type = res.get('resourceType', '')
        res_name = res.get('resourceName', '')
        pattern_type = res.get('resourcePatternType', 'LITERAL')

        for rule in res.get('acls', []):
            counter += 1
            principal = rule.get('principal', '')
            host = rule.get('host', '*')
            operation = rule.get('operation', '')
            permission = rule.get('permissionType', '')

            safe_name = f'acl_{counter}'

            resource = f'''resource "axonops_kafka_acl" "{safe_name}" {{
  cluster_name          = "{cluster_name}"
  resource_type         = "{res_type}"
  resource_name         = "{res_name}"
  resource_pattern_type = "{pattern_type}"
  principal             = "{principal}"
  host                  = "{host}"
  operation             = "{operation}"
  permission_type       = "{permission}"
}}
'''
            resources.append(resource)
            imports.append(
                f'terraform import axonops_kafka_acl.{safe_name} '
                f'"{cluster_name}/{res_type}/{res_name}/{pattern_type}/{principal}/{host}/{operation}/{permission}"'
            )

    if resources:
        with open(os.path.join(output_dir, 'acls.tf'), 'w') as f:
            f.write('# Kafka ACLs\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} ACLs")

    return imports


def import_schemas(api_base: str, api_key: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import schemas and return import commands."""
    print("Fetching schemas...")
    url = f"{api_base}/kafka/{cluster_name}/registry/subjects"
    data = make_request(url, api_key)

    if not data:
        print("  No schemas found")
        return []

    # API returns {"subjects": [...]}
    subjects = data.get('subjects', []) if isinstance(data, dict) else data
    if not subjects:
        print("  No schemas found")
        return []

    resources = []
    imports = []

    for subject in subjects:
        safe_name = sanitize_name(subject)

        # Get the latest schema version
        schema_url = f"{api_base}/kafka/{cluster_name}/registry/subjects/{subject}/latest?getConfigs=true"
        schema_info = make_request(schema_url, api_key)

        if not schema_info:
            continue

        schema_type = schema_info.get('type', 'AVRO')
        schema_str = schema_info.get('schema', '{}')

        # Escape the schema string for HCL
        schema_escaped = schema_str.replace('\\', '\\\\').replace('"', '\\"')

        resource = f'''resource "axonops_schema" "{safe_name}" {{
  cluster_name = "{cluster_name}"
  subject      = "{subject}"
  schema_type  = "{schema_type}"
  schema       = "{schema_escaped}"
}}
'''
        resources.append(resource)
        imports.append(f'terraform import axonops_schema.{safe_name} "{cluster_name}/{subject}"')

    if resources:
        with open(os.path.join(output_dir, 'schemas.tf'), 'w') as f:
            f.write('# Schema Registry Schemas\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} schemas")

    return imports


def import_connectors(api_base: str, api_key: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import connectors and return import commands."""
    print("Fetching connectors...")

    # First, get the list of connect clusters
    clusters_url = f"{api_base}/kafka/{cluster_name}/connect/clusters/"
    connect_clusters = make_request(clusters_url, api_key)

    if not connect_clusters or not isinstance(connect_clusters, list):
        print("  No connect clusters found")
        return []

    resources = []
    imports = []

    for connect_cluster in connect_clusters:
        connect_cluster_name = connect_cluster.get('clusterName', '')
        if not connect_cluster_name:
            continue

        # Get connectors for this connect cluster
        connectors_url = f"{api_base}/kafka/{cluster_name}/connect/{connect_cluster_name}/connectors"
        connectors_data = make_request(connectors_url, api_key)

        if not connectors_data:
            continue

        connectors = connectors_data.get('connectors', {})
        if not connectors:
            continue

        for connector_name, connector_info in connectors.items():
            safe_name = sanitize_name(f"{connect_cluster_name}_{connector_name}")

            # Get connector config
            info = connector_info.get('info', {})
            config = info.get('config', {})

            # Build config map for Terraform (includes connector.class)
            config_lines = []
            for key, value in config.items():
                # Skip name as it's a separate attribute
                if key == 'name':
                    continue
                config_lines.append(f'    "{escape_hcl_string(key)}" = "{escape_hcl_string(str(value))}"')

            resource = f'''resource "axonops_kafka_connect_connector" "{safe_name}" {{
  cluster_name         = "{cluster_name}"
  connect_cluster_name = "{connect_cluster_name}"
  name                 = "{escape_hcl_string(connector_name)}"
  config = {{
{chr(10).join(config_lines)}
  }}
}}
'''

            resources.append(resource)
            imports.append(
                f'terraform import axonops_kafka_connect_connector.{safe_name} '
                f'"{cluster_name}/{connect_cluster_name}/{connector_name}"'
            )

    if resources:
        with open(os.path.join(output_dir, 'connectors.tf'), 'w') as f:
            f.write('# Kafka Connectors\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} connectors")
    else:
        print("  No connectors found")

    return imports


def import_log_collectors(api_base: str, api_key: str, org_id: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import log collectors and return import commands."""
    print("Fetching log collectors...")
    # Note: logcollectors endpoint uses a different path pattern
    url = f"{api_base.rsplit('/', 1)[0]}/logcollectors/{org_id}/kafka/{cluster_name}"
    collectors = make_request(url, api_key)

    if not collectors or not isinstance(collectors, list):
        print("  No log collectors found")
        return []

    resources = []
    imports = []

    for lc in collectors:
        name = lc.get('name', '')
        safe_name = sanitize_name(name)
        filename = escape_hcl_string(lc.get('filename', ''))
        date_format = escape_hcl_string(lc.get('dateFormat', 'yyyy-MM-dd HH:mm:ss,SSS'))
        agent_types = lc.get('supportedAgentType', ['all'])
        info_regex = escape_hcl_string(lc.get('infoRegex', ''))
        warning_regex = escape_hcl_string(lc.get('warningRegex', ''))
        error_regex = escape_hcl_string(lc.get('errorRegex', ''))
        debug_regex = escape_hcl_string(lc.get('debugRegex', ''))
        error_threshold = lc.get('errorAlertThreshold', 0)

        agent_types_str = ', '.join([f'"{t}"' for t in agent_types])

        resource = f'''resource "axonops_logcollector" "{safe_name}" {{
  cluster_name          = "{cluster_name}"
  name                  = "{escape_hcl_string(name)}"
  filename              = "{filename}"
  date_format           = "{date_format}"
  supported_agent_types = [{agent_types_str}]'''

        if info_regex:
            resource += f'\n  info_regex            = "{info_regex}"'
        if warning_regex:
            resource += f'\n  warning_regex         = "{warning_regex}"'
        if error_regex:
            resource += f'\n  error_regex           = "{error_regex}"'
        if debug_regex:
            resource += f'\n  debug_regex           = "{debug_regex}"'
        if error_threshold:
            resource += f'\n  error_alert_threshold = {error_threshold}'

        resource += '\n}\n'

        resources.append(resource)
        imports.append(f'terraform import axonops_logcollector.{safe_name} "{cluster_name}/{name}"')

    if resources:
        with open(os.path.join(output_dir, 'logcollectors.tf'), 'w') as f:
            f.write('# Log Collectors\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} log collectors")

    return imports


def import_healthchecks(api_base: str, api_key: str, org_id: str, cluster_name: str, output_dir: str) -> list[str]:
    """Import healthchecks and return import commands."""
    print("Fetching healthchecks...")
    # Note: healthchecks endpoint uses a different path pattern
    url = f"{api_base.rsplit('/', 1)[0]}/healthchecks/{org_id}/kafka/{cluster_name}"
    data = make_request(url, api_key)

    if not data:
        print("  No healthchecks found")
        return []

    resources = []
    imports = []

    # TCP Healthchecks
    for hc in data.get('tcpchecks', []):
        name = hc.get('name', '')
        safe_name = 'tcp_' + sanitize_name(name)
        tcp = escape_hcl_string(hc.get('tcp', ''))
        interval = hc.get('interval', '1m')
        timeout = hc.get('timeout', '1m')
        readonly = hc.get('readonly', False)
        agent_types = hc.get('supportedAgentType', [])

        agent_types_str = ', '.join([f'"{t}"' for t in agent_types]) if agent_types else '"all"'

        resource = f'''resource "axonops_healthcheck_tcp" "{safe_name}" {{
  cluster_name          = "{cluster_name}"
  name                  = "{escape_hcl_string(name)}"
  tcp                   = "{tcp}"
  interval              = "{interval}"
  timeout               = "{timeout}"
  readonly              = {str(readonly).lower()}
  supported_agent_types = [{agent_types_str}]
}}
'''
        resources.append(resource)
        imports.append(f'terraform import axonops_healthcheck_tcp.{safe_name} "{cluster_name}/{name}"')

    # HTTP Healthchecks
    for hc in data.get('httpchecks', []):
        name = hc.get('name', '')
        safe_name = 'http_' + sanitize_name(name)
        url_val = escape_hcl_string(hc.get('url', ''))
        method = hc.get('method', 'GET')
        interval = hc.get('interval', '1m')
        timeout = hc.get('timeout', '1m')
        expected_status = hc.get('expectedStatus', 200)
        readonly = hc.get('readonly', False)
        body = escape_hcl_string(hc.get('body', ''))
        headers = hc.get('headers', {})
        agent_types = hc.get('supportedAgentType', [])

        agent_types_str = ', '.join([f'"{t}"' for t in agent_types]) if agent_types else '"all"'

        resource = f'''resource "axonops_healthcheck_http" "{safe_name}" {{
  cluster_name          = "{cluster_name}"
  name                  = "{escape_hcl_string(name)}"
  url                   = "{url_val}"
  method                = "{method}"
  expected_status       = {expected_status}
  interval              = "{interval}"
  timeout               = "{timeout}"
  readonly              = {str(readonly).lower()}
  supported_agent_types = [{agent_types_str}]'''

        if body:
            resource += f'\n  body                  = "{body}"'

        if headers:
            headers_str = '\n    '.join([f'"{escape_hcl_string(k)}" = "{escape_hcl_string(v)}"' for k, v in headers.items()])
            resource += f'\n  headers = {{\n    {headers_str}\n  }}'

        resource += '\n}\n'

        resources.append(resource)
        imports.append(f'terraform import axonops_healthcheck_http.{safe_name} "{cluster_name}/{name}"')

    # Shell Healthchecks
    for hc in data.get('shellchecks', []):
        name = hc.get('name', '')
        safe_name = 'shell_' + sanitize_name(name)
        script = escape_hcl_string(hc.get('script', ''))
        shell = escape_hcl_string(hc.get('shell', ''))
        interval = hc.get('interval', '1m')
        timeout = hc.get('timeout', '1m')
        readonly = hc.get('readonly', False)

        resource = f'''resource "axonops_healthcheck_shell" "{safe_name}" {{
  cluster_name = "{cluster_name}"
  name         = "{escape_hcl_string(name)}"
  script       = "{script}"
  interval     = "{interval}"
  timeout      = "{timeout}"
  readonly     = {str(readonly).lower()}'''

        if shell:
            resource += f'\n  shell        = "{shell}"'

        resource += '\n}\n'

        resources.append(resource)
        imports.append(f'terraform import axonops_healthcheck_shell.{safe_name} "{cluster_name}/{name}"')

    if resources:
        with open(os.path.join(output_dir, 'healthchecks.tf'), 'w') as f:
            f.write('# Healthchecks\n\n')
            f.write('\n'.join(resources))
        print(f"  Found {len(resources)} healthchecks")

    return imports


def main():
    parser = argparse.ArgumentParser(
        description='Import AxonOps Kafka cluster resources into Terraform',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='''
Example:
    python import-cluster.py axonops.example.com:8080 myorg mycluster abc123 ./imported

Environment variables:
    PROTOCOL    - Protocol to use (http or https, default: http)
'''
    )
    parser.add_argument('axonops_host', help='AxonOps server hostname (e.g., axonops.example.com:8080)')
    parser.add_argument('org_id', help='Organization ID')
    parser.add_argument('cluster_name', help='Kafka cluster name')
    parser.add_argument('api_key', help='API key for authentication')
    parser.add_argument('output_dir', nargs='?', default='./imported', help='Output directory (default: ./imported)')

    args = parser.parse_args()

    protocol = os.environ.get('PROTOCOL', 'http')
    api_base = f"{protocol}://{args.axonops_host}/api/v1/{args.org_id}"

    # Create output directory
    os.makedirs(args.output_dir, exist_ok=True)

    print("AxonOps Kafka Terraform Import Script")
    print("=" * 40)
    print(f"Host: {args.axonops_host}")
    print(f"Org: {args.org_id}")
    print(f"Cluster: {args.cluster_name}")
    print(f"Output: {args.output_dir}")
    print()

    # Generate provider configuration
    print("Generating provider configuration...")
    generate_provider_tf(args.output_dir, args.axonops_host, protocol, args.org_id)

    # Collect all import commands
    all_imports = []

    # Import resources
    all_imports.extend(import_topics(api_base, args.api_key, args.cluster_name, args.output_dir))
    all_imports.extend(import_acls(api_base, args.api_key, args.cluster_name, args.output_dir))
    all_imports.extend(import_schemas(api_base, args.api_key, args.cluster_name, args.output_dir))
    all_imports.extend(import_connectors(api_base, args.api_key, args.cluster_name, args.output_dir))
    all_imports.extend(import_log_collectors(api_base, args.api_key, args.org_id, args.cluster_name, args.output_dir))
    all_imports.extend(import_healthchecks(api_base, args.api_key, args.org_id, args.cluster_name, args.output_dir))

    # Write import commands script
    import_file = os.path.join(args.output_dir, 'import_commands.sh')
    with open(import_file, 'w') as f:
        f.write('#!/bin/bash\n')
        f.write(f'# Terraform import commands for cluster: {args.cluster_name}\n')
        f.write(f'# Generated by import-cluster.py\n\n')
        f.write('set -e\n\n')
        f.write('# Check that API key is set\n')
        f.write('if [ -z "$TF_VAR_axonops_api_key" ]; then\n')
        f.write('  echo "Error: TF_VAR_axonops_api_key environment variable is not set"\n')
        f.write('  echo "Run: export TF_VAR_axonops_api_key=\'your-api-key\'"\n')
        f.write('  exit 1\n')
        f.write('fi\n\n')

        if all_imports:
            f.write('\n'.join(all_imports))
            f.write('\n')

    os.chmod(import_file, 0o755)

    print()
    print("=" * 40)
    print("Import complete!")
    print()
    print(f"Generated files in {args.output_dir}:")
    for filename in sorted(os.listdir(args.output_dir)):
        filepath = os.path.join(args.output_dir, filename)
        size = os.path.getsize(filepath)
        print(f"  {filename} ({size} bytes)")
    print()
    print("Next steps:")
    print("1. Review the generated .tf files")
    print("2. Set your API key: export TF_VAR_axonops_api_key='your-api-key'")
    print("3. Initialize Terraform: terraform init")
    print(f"4. Run the import commands: bash {import_file}")
    print("5. Verify the state: terraform plan")
    print()


if __name__ == '__main__':
    main()
