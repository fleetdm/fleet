'use strict';

const util = require('util');
const execFile = util.promisify(require('child_process').execFile);

const EXEC_OPTIONS = { timeout: 10000, maxBuffer: 10 * 1024 * 1024 };

const TABLES = [
  'api_resources',
  'cluster_role_binding_subjects',
  'cluster_role_policy_rules',
  'component_statuses',
  'config_maps',
  'cron_jobs',
  'csi_drivers',
  'csi_node_drivers',
  'daemon_set_containers',
  'daemon_set_volumes',
  'daemon_sets',
  'deployments',
  'deployments_containers',
  'deployments_volumes',
  'endpoint_subsets',
  'events',
  'horizontal_pod_autoscalers',
  'info',
  'ingress_classes',
  'ingresses',
  'jobs',
  'limit_ranges',
  'mutating_webhooks',
  'namespaces',
  'network_policies',
  'nodes',
  'persistent_volume_claims',
  'persistent_volumes',
  'pod_containers',
  'pod_disruption_budgets',
  'pod_security_policies',
  'pod_template_containers',
  'pod_templates',
  'pod_templates_volumes',
  'pod_volumes',
  'pods',
  'replica_set_containers',
  'replica_set_volumes',
  'replica_sets',
  'resource_quotas',
  'role_binding_subjects',
  'role_policy_rules',
  'secrets',
  'service_accounts',
  'services',
  'stateful_set_containers',
  'stateful_set_volumes',
  'stateful_sets',
  'storage_classes',
  'validating_webhooks',
  'volume_attachments'
];

async function getPodName() {
  const { stdout, stderr } = await execFile('kubectl', ['get', 'pods', '-n', 'kubequery', '-o', "jsonpath={.items[0].metadata.name}"], EXEC_OPTIONS);
  if (stdout) {
    return stdout;
  }
  throw new Error('Failed to get kubequery pod name');
}

async function executeSQL(podName, sql) {
  const { stdout, stderr } = await execFile('kubectl', ['exec', '-it', podName, '-n', 'kubequery', '--', 'sh', '-c', `kubequeryi --json '${sql}'`], EXEC_OPTIONS);
  if (stdout) {
    return stdout;
  }
  throw new Error('Failed to execute SQL: ' + sql + '. Error: ' + stderr);
}

(async () => {
  try {
    const podName = await getPodName();
    for (const table of TABLES) {
      const output = await executeSQL(podName, 'SELECT * FROM kubernetes_' + table);
      console.assert(output !== '', 'Invalid output for table: ' + table);

      const json = JSON.parse(output);
      console.assert(Array.isArray(json), 'Table output is not an array: ' + table);

      console.info(table + ': ' + json.length);
    }
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
})();
