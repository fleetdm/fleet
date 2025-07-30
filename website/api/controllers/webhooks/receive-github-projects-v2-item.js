// Dependencies
const {BigQuery} = require('@google-cloud/bigquery');
const crypto = require('crypto');

// Project constants
const PROJECTS = {
  ORCHESTRATION: 1,
  MDM: 2,
  SOFTWARE: 3
};

// BigQuery client (initialized on first use)
let bigqueryClient = null;

module.exports = {

  friendlyName: 'Receive GitHub Projects v2 item',

  description: 'Receive webhook requests from GitHub for Projects v2 item events.',

  extendedDescription: 'This webhook endpoint receives JSON data from GitHub when project items are created, updated, or deleted.',

  inputs: {
    // GitHub sends the entire payload as JSON in the request body
    // We'll capture all the data dynamically
  },

  exits: {
    success: {
      description: 'Webhook processed successfully'
    },
    unauthorized: {
      description: 'Invalid or missing webhook signature',
      responseType: 'unauthorized'
    }
  },

  fn: async function () {

    // Check if webhook secret is configured
    if (!sails.config.custom.githubProjectsV2ItemWebhookSecret) {
      throw new Error('No GitHub Projects v2 item webhook secret configured! (Please set `sails.config.custom.githubProjectsV2ItemWebhookSecret`.)');
    }

    // Verify webhook signature
    verifyGitHubWebhookSignature(this.req, sails.config.custom.githubProjectsV2ItemWebhookSecret);

    // Get event type from headers
    const eventType = this.req.get('X-GitHub-Event');
    if (eventType !== 'projects_v2_item') {
      sails.log.warn('Unexpected event type:', eventType);
      return {
        success: true,
        message: 'Webhook received but unexpected event type'
      };
    }

    // Check and parse GCP service account key
    let gcpServiceAccountKey = parseGcpServiceAccountKey();
    if (!gcpServiceAccountKey) {
      // Error already logged in parseGcpServiceAccountKey function
      return {
        success: true,
        message: 'Webhook received but GCP configuration error'
      };
    }

    // Check if we have the required GitHub access token
    if (!sails.config.custom.githubAccessToken) {
      sails.log.error('No GitHub access token configured for fetching issue details');
      return {
        success: true,
        message: 'Webhook received but GitHub access token missing'
      };
    }

    // Validations passed, process the webhook
    let payload = typeof this.req.body === 'string' ? JSON.parse(this.req.body) : this.req.body;

    // TODO: remove this
    // Pretty print the JSON to the console
    console.log('\n========================================');
    console.log('GitHub Projects v2 Item Webhook');
    console.log('========================================');
    console.log('Timestamp:', new Date().toISOString());
    console.log('Headers:', JSON.stringify(this.req.headers, null, 2));
    console.log('Payload:', JSON.stringify(payload, null, 2));
    console.log('========================================\n');

    // Process the webhook data
    try {

      // Check if this is a status change we care about
      const statusChange = await processStatusChange(payload, gcpServiceAccountKey);

      if (statusChange) {
        sails.log.info('Processed issue status change:', statusChange);
      }
    } catch (err) {
      sails.log.error('Error processing webhook:', err);
      // Return success to GitHub even if processing fails
      return {
        success: true,
        message: 'Webhook received but processing error occurred'
      };
    }

    // Return success response
    return {
      success: true,
      message: 'Webhook received and processed successfully'
    };

  }

};

/**
 * Verifies GitHub webhook signature
 *
 * @param {Object} req - The request object
 * @param {string} secret - The webhook secret
 * @throws {string} 'unauthorized' if signature is invalid or missing
 */
function verifyGitHubWebhookSignature(req, secret) {
  // Check if webhook secret is configured
  if (!secret) {
    throw new Error('No GitHub webhook secret configured!');
  }

  // Get the signature from the header
  const signature = req.get('X-Hub-Signature-256');

  if (!signature) {
    sails.log.warn('GitHub webhook received without X-Hub-Signature-256 header');
    throw 'unauthorized';
  }

  // Get the request body for signature verification
  // Stringify the parsed body to compute the signature
  let rawBody = typeof req.body === 'string' ? req.body : JSON.stringify(req.body);

  // Create HMAC hash
  const hmac = crypto.createHmac('sha256', secret);
  const digest = 'sha256=' + hmac.update(rawBody, 'utf8').digest('hex');

  // Compare signatures using timing-safe comparison
  const signatureBuffer = Buffer.from(signature);
  const digestBuffer = Buffer.from(digest);

  // Check if buffers have the same length and use timing-safe comparison
  if (signatureBuffer.length !== digestBuffer.length || !crypto.timingSafeEqual(signatureBuffer, digestBuffer)) {
    sails.log.warn('GitHub webhook received with invalid signature');
    throw 'unauthorized';
  }
}

/**
 * Parses and validates GCP service account key from configuration
 *
 * @returns {Object|null} Parsed GCP service account key object or null if invalid/missing
 */
function parseGcpServiceAccountKey() {
  if (!sails.config.custom.engMetricsGcpServiceAccountKey) {
    sails.log.error('No GCP service account key configured for engineering metrics');
    return null;
  }

  try {
    let gcpServiceAccountKey;

    // Check if it's already an object or needs parsing
    if (typeof sails.config.custom.engMetricsGcpServiceAccountKey === 'object') {
      gcpServiceAccountKey = sails.config.custom.engMetricsGcpServiceAccountKey;
    } else if (typeof sails.config.custom.engMetricsGcpServiceAccountKey === 'string') {
      // Fix common JSON formatting issues before parsing
      let jsonString = sails.config.custom.engMetricsGcpServiceAccountKey;

      // Replace actual newlines within the JSON string with escaped newlines
      // This handles cases where the private key has literal newlines
      jsonString = jsonString.replace(/\n/g, '\\n');

      // Parse the cleaned JSON
      gcpServiceAccountKey = JSON.parse(jsonString);
    } else {
      throw new Error('Invalid GCP service account key type');
    }

    // Validate that it has the expected structure
    if (!gcpServiceAccountKey.type || !gcpServiceAccountKey.project_id || !gcpServiceAccountKey.private_key) {
      throw new Error('Invalid GCP service account key structure');
    }

    return gcpServiceAccountKey;
  } catch (err) {
    sails.log.error('Failed to parse GCP service account key:', err);
    return null;
  }
}

/**
 * Processes status changes from GitHub Projects v2 webhook
 *
 * @param {Object} payload - The webhook payload
 * @param {Object} gcpServiceAccountKey - The parsed GCP service account key
 * @returns {Object|null} Status change data if it should be saved, null otherwise
 */
async function processStatusChange(payload, gcpServiceAccountKey) {
  // Check if this is a project item update with status change
  if (!payload.changes || !payload.changes.field_value) {
    return null;
  }

  const fieldValue = payload.changes.field_value;

  // Check if this is a status field change
  if (fieldValue.field_name !== 'Status') {
    return null;
  }

  // Check if this is one of our tracked projects
  const projectNumber = fieldValue.project_number;
  const validProjects = Object.values(PROJECTS);

  if (!validProjects.includes(projectNumber)) {
    sails.log.verbose(`Ignoring status change for project ${projectNumber} - not a tracked project`);
    return null;
  }

  // Check if status changed to "in progress" from "ready" or null
  // from and to are either objects with a name property or null
  const fromStatus = fieldValue.from ? fieldValue.from.name.toLowerCase() : '';
  const toStatus = fieldValue.to ? fieldValue.to.name.toLowerCase() : '';

  // Log the status change for debugging
  sails.log.verbose(`Status change detected: "${fromStatus || '(null)'}" -> "${toStatus}"`);

  // Check if the "to" status includes "in progress"
  if (!toStatus.includes('in progress')) {
    sails.log.verbose(`Ignoring status change - "to" status doesn't include "in progress": ${toStatus}`);
    return null;
  }

  // Check if the "from" status is null or includes "ready"
  const isFromNullOrReady = fieldValue.from === null || fromStatus.includes('ready');

  if (!isFromNullOrReady) {
    sails.log.verbose(`Status change from "${fromStatus}" to "in progress" - will check if already tracked`);
  }

  // Get issue details from the payload
  const projectsV2Item = payload.projects_v2_item;
  if (!projectsV2Item || !projectsV2Item.content_node_id) {
    sails.log.error('Missing projects_v2_item or content_node_id in payload');
    return null;
  }

  // Fetch issue details from GitHub API
  const issueDetails = await fetchIssueDetails(projectsV2Item.content_node_id);
  if (!issueDetails) {
    return null;
  }

  // If the "from" status is not null or ready, check if we already have this issue tracked
  if (!isFromNullOrReady) {
    const exists = await checkIfIssueExists(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);
    if (exists) {
      sails.log.verbose(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} already tracked as in_progress, skipping`);
      return null;
    }
    sails.log.info(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} not yet tracked, will save as in_progress`);
  }

  // Prepare data for BigQuery
  // We already verified the status includes "in progress", so save it as "in_progress" in the DB
  const statusChangeData = {
    date: new Date().toISOString(),
    repo: issueDetails.repo,
    issue_number: issueDetails.issueNumber,
    status: 'in_progress'
  };

  // Save to BigQuery
  await saveToBigQuery(statusChangeData, gcpServiceAccountKey);

  return statusChangeData;
}


/**
 * Fetches issue details from GitHub API using the node ID
 *
 * @param {string} nodeId - The GitHub node ID of the issue
 * @returns {Object|null} Issue details (repo and issue number) or null if error
 */
async function fetchIssueDetails(nodeId) {
  try {
    // GitHub GraphQL API query to get issue details from node ID
    const query = `
      query($nodeId: ID!) {
        node(id: $nodeId) {
          ... on Issue {
            number
            repository {
              nameWithOwner
            }
          }
        }
      }
    `;

    const response = await sails.helpers.http.post('https://api.github.com/graphql', {
      query: query,
      variables: { nodeId: nodeId }
    }, {
      'Authorization': `Bearer ${sails.config.custom.githubAccessToken}`,
      'Accept': 'application/vnd.github.v4+json',
      'User-Agent': 'Fleet-Engineering-Metrics'
    });

    if (!response.data || !response.data.node) {
      sails.log.error('No data returned from GitHub API for node:', nodeId);
      return null;
    }

    const node = response.data.node;
    return {
      repo: node.repository.nameWithOwner,
      issueNumber: node.number
    };
  } catch (err) {
    sails.log.error('Error fetching issue details from GitHub:', err);
    return null;
  }
}

/**
 * Gets or initializes the BigQuery client (singleton pattern)
 *
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @returns {BigQuery} The BigQuery client instance
 */
function getBigQueryClient(gcpServiceAccountKey) {
  if (!bigqueryClient) {
    bigqueryClient = new BigQuery({
      projectId: gcpServiceAccountKey.project_id,
      credentials: gcpServiceAccountKey
    });
  }
  return bigqueryClient;
}

/**
 * Checks if an issue already exists in BigQuery with in_progress status
 *
 * @param {string} repo - The repository name (e.g., "fleetdm/fleet")
 * @param {number} issueNumber - The issue number
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @returns {boolean} True if the issue exists, false otherwise
 */
async function checkIfIssueExists(repo, issueNumber, gcpServiceAccountKey) {
  try {
    // Get BigQuery client
    const bigquery = getBigQueryClient(gcpServiceAccountKey);

    // Configure dataset and table names
    const datasetId = 'github_metrics';
    const tableId = 'issue_status_change';

    // Query to check if the issue exists with in_progress status
    const query = `
      SELECT COUNT(*) as count
      FROM \`${gcpServiceAccountKey.project_id}.${datasetId}.${tableId}\`
      WHERE repo = @repo
        AND issue_number = @issueNumber
        AND status = 'in_progress'
      LIMIT 1
    `;

    const options = {
      query: query,
      params: {
        repo: repo,
        issueNumber: issueNumber
      }
    };

    // Run the query
    const [rows] = await bigquery.query(options);

    return rows[0].count > 0;
  } catch (err) {
    sails.log.error('Error checking if issue exists in BigQuery:', err);
    // On error, assume it doesn't exist to avoid blocking new records
    return false;
  }
}

/**
 * Saves status change data to BigQuery
 *
 * @param {Object} data - The status change data to save
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 */
async function saveToBigQuery(data, gcpServiceAccountKey) {
  try {
    // Get BigQuery client
    const bigquery = getBigQueryClient(gcpServiceAccountKey);

    // Configure dataset and table names
    const datasetId = 'github_metrics';
    const tableId = 'issue_status_change';

    // Get reference to the table
    const dataset = bigquery.dataset(datasetId);
    const table = dataset.table(tableId);

    // Insert the data
    await table.insert([data]);

    sails.log.debug('Successfully saved status change to BigQuery:', {
      dataset: datasetId,
      table: tableId,
      data: data
    });

  } catch (err) {
    // If we get a connection error, reset the client so it will be recreated on next attempt
    if (err.code === 'ECONNREFUSED' || err.code === 'ETIMEDOUT' || err.code === 'ENOTFOUND') {
      sails.log.warn('BigQuery connection error detected, resetting client:', err.code);
      bigqueryClient = null;
    }
    // Handle specific BigQuery errors
    if (err.name === 'PartialFailureError') {
      // Log the specific rows that failed
      sails.log.error('Partial failure when inserting to BigQuery:', err.errors);
    } else if (err.code === 404) {
      sails.log.error('BigQuery table or dataset not found. Please ensure the table exists:', {
        dataset: 'engineering_metrics',
        table: 'project_status_changes'
      });
    } else if (err.code === 403) {
      sails.log.error('Permission denied when accessing BigQuery. Check service account permissions.');
    } else {
      sails.log.error('Error saving to BigQuery:', err);
    }

    throw err;
  }
}
