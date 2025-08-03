/**
 * GitHub Projects v2 status change tracking
 *
 * This webhook tracks issue status changes in GitHub Projects v2 for engineering metrics.
 *
 * Tracked projects:
 * - Orchestration
 * - MDM
 * - Software
 *
 * Status transitions tracked:
 *
 * 1. "In progress" status:
 *    - Triggers when: Status changes TO "in progress"
 *    - From states: "ready" or null (first time) OR any other state (if not already tracked)
 *    - Saves to: github_metrics.issue_status_change
 *    - Data: timestamp, repo, issue_number, status = 'in_progress'
 *
 * 2. "Awaiting QA" status:
 *    - Triggers when: Status changes TO "awaiting qa"
 *    - From states:
 *      - "in progress" or "review" → Always creates new row
 *      - Any other state → Creates row only if no QA ready row exists
 *    - Saves to: github_metrics.issue_qa_ready
 *    - Data: qa_ready time, assignee, issue details, time from in_progress to qa_ready
 *    - Note: Requires an existing in_progress record to calculate time
 *
 * 3. "Release" status:
 *    - Triggers when: Status changes TO "release"
 *    - From states:
 *      - "awaiting qa" → Always creates new row
 *      - Any other state → Creates row only if issue has been in QA (exists in issue_qa_ready)
 *    - Saves to: github_metrics.issue_release_ready
 *    - Data: release_ready time, assignee, issue details, time from in_progress to release_ready
 *    - Note: Requires an existing in_progress record to calculate time
 *
 * Time calculations:
 * - All time calculations can optionally exclude weekends (controlled by EXCLUDE_WEEKENDS flag)
 * - Weekend exclusion adjusts start/end times and subtracts weekend days from duration
 * - Times are calculated from the webhook's updated_at timestamp for accuracy
 *
 * Issue type classification:
 * - Based on GitHub issue labels:
 *   - "bug" → type: "bug"
 *   - "story" → type: "story"
 *   - "~sub-task" → type: "sub-task"
 *   - Otherwise → type: "other"
 */

// Dependencies
const {BigQuery} = require('@google-cloud/bigquery');
const crypto = require('crypto');

// Project constants
const PROJECTS = {
  ORCHESTRATION: 71,
  MDM: 58,
  SOFTWARE: 70
};

// Flag for excluding weekends in time calculations. Only used during dev.
// Set to true to exclude weekends, false to include them
const EXCLUDE_WEEKENDS = true;

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

      // This handles cases where the private key has literal newlines
      jsonString = jsonString.replace(/"private_key":\s*"([^"]+)"/g, (match, key) => {
        // Replace actual newlines with escaped newlines only within the private key value
        const fixedKey = key.replace(/\n/g, '\\n');
        return `"private_key": "${fixedKey}"`;
      });

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

  // Check if the "to" status includes "in progress", "awaiting qa", or "release"
  const isToInProgress = toStatus.includes('in progress');
  const isToAwaitingQa = toStatus.includes('awaiting qa');
  const isToRelease = toStatus.includes('release');

  if (!isToInProgress && !isToAwaitingQa && !isToRelease) {
    sails.log.verbose(`Ignoring status change - "to" status doesn't include "in progress", "awaiting qa", or "release": ${toStatus}`);
    return null;
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

  // Handle "in progress" status changes
  if (isToInProgress) {
    return await handleInProgressStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey);
  }

  // Handle "awaiting qa" status changes
  if (isToAwaitingQa) {
    return await handleAwaitingQaStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey, projectNumber);
  }

  // Handle "release" status changes
  if (isToRelease) {
    return await handleReleaseStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey, projectNumber);
  }

  return null;
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
            assignees(first: 1) {
              nodes {
                login
              }
            }
            labels(first: 20) {
              nodes {
                name
              }
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
    const assignee = node.assignees.nodes.length > 0 ? node.assignees.nodes[0].login : '';

    // Extract label names
    const labels = node.labels.nodes.map(label => label.name.toLowerCase());

    // Determine issue type based on labels
    let issueType = 'other';
    if (labels.includes('bug')) {
      issueType = 'bug';
    } else if (labels.includes('story')) {
      issueType = 'story';
    } else if (labels.includes('~sub-task')) {
      issueType = 'sub-task';
    }

    return {
      repo: node.repository.nameWithOwner,
      issueNumber: node.number,
      assignee: assignee,
      type: issueType
    };
  } catch (err) {
    sails.log.error('Error fetching issue details from GitHub:', err);
    return null;
  }
}

/**
 * Handles status changes to "in progress"
 *
 * @param {Object} fieldValue - The field value from the webhook payload
 * @param {Object} projectsV2Item - The project item from the webhook payload
 * @param {Object} issueDetails - The issue details from GitHub API
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @returns {Object|null} Status change data if saved, null otherwise
 */
async function handleInProgressStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey) {
  // Check if the "from" status is null or includes "ready"
  const fromStatus = fieldValue.from ? fieldValue.from.name.toLowerCase() : '';
  const isFromNullOrReady = fieldValue.from === null || fromStatus.includes('ready');

  if (!isFromNullOrReady) {
    sails.log.verbose(`Status change from "${fromStatus}" to "in progress" - will check if already tracked`);
    const exists = await checkIfIssueExists(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);
    if (exists) {
      sails.log.verbose(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} already tracked as in_progress, skipping`);
      return null;
    }
    sails.log.info(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} not yet tracked, will save as in_progress`);
  }

  // Prepare data for BigQuery
  const statusChangeData = {
    date: projectsV2Item.updated_at,  // Use the actual update time from webhook
    repo: issueDetails.repo,
    issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
    status: 'in_progress'
  };

  // Save to BigQuery
  await saveToBigQuery(statusChangeData, gcpServiceAccountKey);
  return statusChangeData;
}

/**
 * Handles status changes to "awaiting qa"
 *
 * @param {Object} fieldValue - The field value from the webhook payload
 * @param {Object} projectsV2Item - The project item from the webhook payload
 * @param {Object} issueDetails - The issue details from GitHub API
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @param {number} projectNumber - The project number
 * @returns {Object|null} QA ready data if saved, null otherwise
 */
async function handleAwaitingQaStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey, projectNumber) {
  // Check if from status is "in progress" or "review"
  const fromStatus = fieldValue.from ? fieldValue.from.name.toLowerCase() : '';
  const isFromInProgressOrReview = fromStatus.includes('in progress') || fromStatus.includes('review');

  // Check if we should create a new QA ready row
  let shouldCreateQaRow = false;

  if (isFromInProgressOrReview) {
    // Always create if transitioning from in progress or review
    shouldCreateQaRow = true;
  } else {
    // Check if row already exists
    const qaRowExists = await checkIfQaReadyExists(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);
    if (!qaRowExists) {
      shouldCreateQaRow = true;
    } else {
      sails.log.verbose(`QA ready row already exists for ${issueDetails.repo}#${issueDetails.issueNumber}, skipping`);
      return null;
    }
  }

  if (shouldCreateQaRow) {
    // Get the latest in_progress status from BigQuery
    const inProgressData = await getLatestInProgressStatus(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);

    if (!inProgressData) {
      sails.log.warn(`No in_progress status found for ${issueDetails.repo}#${issueDetails.issueNumber}, cannot calculate QA ready time`);
      return null;
    }

    // Calculate time to QA ready
    const qaReadyTime = new Date(projectsV2Item.updated_at);  // Use webhook timestamp
    const inProgressTime = new Date(inProgressData.date);
    const timeToQaReadySeconds = calculateTimeExcludingWeekends(inProgressTime, qaReadyTime);

    // Determine project name
    let projectName = '';
    switch (projectNumber) {
      case PROJECTS.ORCHESTRATION:
        projectName = 'orchestration';
        break;
      case PROJECTS.MDM:
        projectName = 'mdm';
        break;
      case PROJECTS.SOFTWARE:
        projectName = 'software';
        break;
    }

    // Prepare QA ready data
    const qaReadyData = {
      qa_ready: qaReadyTime.toISOString().split('T')[0],  // eslint-disable-line camelcase
      assignee: issueDetails.assignee || '',  // Get assignee from issue details
      issue_url: `https://github.com/${issueDetails.repo}/issues/${issueDetails.issueNumber}`,  // eslint-disable-line camelcase
      time_to_qa_ready_seconds: timeToQaReadySeconds,  // eslint-disable-line camelcase
      repo: issueDetails.repo,
      issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
      qa_ready_time: qaReadyTime.toISOString(),  // eslint-disable-line camelcase
      in_progress_time: inProgressTime.toISOString(),  // eslint-disable-line camelcase
      project: projectName,
      type: issueDetails.type  // Issue type based on labels
    };

    // Save to BigQuery
    await saveQaReadyToBigQuery(qaReadyData, gcpServiceAccountKey);

    sails.log.info('Saved QA ready metrics:', {
      repo: issueDetails.repo,
      issueNumber: issueDetails.issueNumber,
      timeToQaReadySeconds,
      project: projectName
    });

    return qaReadyData;
  }

  return null;
}

/**
 * Handles status changes to "release"
 *
 * @param {Object} fieldValue - The field value from the webhook payload
 * @param {Object} projectsV2Item - The project item from the webhook payload
 * @param {Object} issueDetails - The issue details from GitHub API
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @param {number} projectNumber - The project number
 * @returns {Object|null} Release ready data if saved, null otherwise
 */
async function handleReleaseStatus(fieldValue, projectsV2Item, issueDetails, gcpServiceAccountKey, projectNumber) {
  // Check if from status is "awaiting qa"
  const fromStatus = fieldValue.from ? fieldValue.from.name.toLowerCase() : '';
  const isFromAwaitingQa = fromStatus.includes('awaiting qa');

  // Check if we should save this release transition
  if (!isFromAwaitingQa) {
    // Not directly from "awaiting qa", check if issue has ever been in QA
    const hasBeenInQa = await checkIfQaReadyExists(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);
    if (!hasBeenInQa) {
      sails.log.verbose(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} has never been in "awaiting qa", skipping release tracking`);
      return null;
    }
    sails.log.info(`Issue ${issueDetails.repo}#${issueDetails.issueNumber} transitioning to release (previously was in QA)`);
  }

  // Always add a new row when transitioning to "release" if it has been through QA
  // (Multiple transitions are allowed and tracked)

  // Get the latest in_progress status from BigQuery
  const inProgressData = await getLatestInProgressStatus(issueDetails.repo, issueDetails.issueNumber, gcpServiceAccountKey);

  if (!inProgressData) {
    sails.log.warn(`No in_progress status found for ${issueDetails.repo}#${issueDetails.issueNumber}, cannot calculate release ready time`);
    return null;
  }

  // Calculate time to release ready (from in_progress to release)
  const releaseReadyTime = new Date(projectsV2Item.updated_at);  // Use webhook timestamp
  const inProgressTime = new Date(inProgressData.date);
  const timeToReleaseReadySeconds = calculateTimeExcludingWeekends(inProgressTime, releaseReadyTime);

  // Determine project name
  let projectName = '';
  switch (projectNumber) {
    case PROJECTS.ORCHESTRATION:
      projectName = 'orchestration';
      break;
    case PROJECTS.MDM:
      projectName = 'mdm';
      break;
    case PROJECTS.SOFTWARE:
      projectName = 'software';
      break;
  }

  // Prepare release ready data
  const releaseReadyData = {
    release_ready: releaseReadyTime.toISOString().split('T')[0],  // eslint-disable-line camelcase
    assignee: issueDetails.assignee || '',  // Get assignee from issue details
    issue_url: `https://github.com/${issueDetails.repo}/issues/${issueDetails.issueNumber}`,  // eslint-disable-line camelcase
    time_to_release_ready_seconds: timeToReleaseReadySeconds,  // eslint-disable-line camelcase
    repo: issueDetails.repo,
    issue_number: issueDetails.issueNumber,  // eslint-disable-line camelcase
    release_ready_time: releaseReadyTime.toISOString(),  // eslint-disable-line camelcase
    in_progress_time: inProgressTime.toISOString(),  // eslint-disable-line camelcase
    project: projectName,
    type: issueDetails.type  // Issue type based on labels
  };

  // Save to BigQuery
  await saveReleaseReadyToBigQuery(releaseReadyData, gcpServiceAccountKey);

  sails.log.info('Saved release ready metrics:', {
    repo: issueDetails.repo,
    issueNumber: issueDetails.issueNumber,
    timeToReleaseReadySeconds,
    project: projectName
  });

  return releaseReadyData;
}

/**
 * Handles common BigQuery errors
 *
 * @param {Error} err - The error object
 * @param {string} operation - Description of the operation that failed
 * @param {string} tableName - The table name for context
 */
function handleBigQueryError(err, operation, tableName) {
  // If we get a connection error, reset the client so it will be recreated on next attempt
  if (err.code === 'ECONNREFUSED' || err.code === 'ETIMEDOUT' || err.code === 'ENOTFOUND') {
    sails.log.warn('BigQuery connection error detected, resetting client:', err.code);
    bigqueryClient = null;
  }
  // Handle specific BigQuery errors
  if (err.name === 'PartialFailureError') {
    // Log the specific rows that failed
    sails.log.error(`Partial failure when ${operation}:`, err.errors);
  } else if (err.code === 404) {
    sails.log.error('BigQuery table or dataset not found. Please ensure the table exists:', {
      dataset: 'github_metrics',
      table: tableName,
      fullError: err.message
    });
  } else if (err.code === 403) {
    sails.log.error('Permission denied when accessing BigQuery. Check service account permissions.');
  } else {
    sails.log.error(`Error ${operation}:`, err);
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
 * Generic helper to check if a record exists in BigQuery
 *
 * @param {string} repo - The repository name
 * @param {number} issueNumber - The issue number
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @param {string} tableId - The table to check
 * @param {string} additionalCondition - Optional additional WHERE clause condition
 * @returns {boolean} True if the record exists, false otherwise
 */
async function checkIfRecordExists(repo, issueNumber, gcpServiceAccountKey, tableId, additionalCondition = '') {
  try {
    const bigquery = getBigQueryClient(gcpServiceAccountKey);
    const datasetId = 'github_metrics';

    const query = `
      SELECT 1
      FROM \`${gcpServiceAccountKey.project_id}.${datasetId}.${tableId}\`
      WHERE repo = @repo
        AND issue_number = @issueNumber
        ${additionalCondition}
      LIMIT 1
    `;

    const options = {
      query: query,
      params: { repo, issueNumber }
    };

    const [rows] = await bigquery.query(options);
    return rows.length > 0;
  } catch (err) {
    if (err.code === 404) {
      return false;
    }
    sails.log.error(`Error checking if record exists in ${tableId}:`, err);
    return false;
  }
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
  return checkIfRecordExists(repo, issueNumber, gcpServiceAccountKey, 'issue_status_change', 'AND status = \'in_progress\'');
}

/**
 * Checks if a QA ready entry already exists in BigQuery
 *
 * @param {string} repo - The repository name (e.g., "fleetdm/fleet")
 * @param {number} issueNumber - The issue number
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @returns {boolean} True if the entry exists, false otherwise
 */
async function checkIfQaReadyExists(repo, issueNumber, gcpServiceAccountKey) {
  return checkIfRecordExists(repo, issueNumber, gcpServiceAccountKey, 'issue_qa_ready');
}

/**
 * Gets the latest in_progress status entry from BigQuery
 *
 * @param {string} repo - The repository name (e.g., "fleetdm/fleet")
 * @param {number} issueNumber - The issue number
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 * @returns {Object|null} The latest in_progress entry or null if not found
 */
async function getLatestInProgressStatus(repo, issueNumber, gcpServiceAccountKey) {
  try {
    // Get BigQuery client
    const bigquery = getBigQueryClient(gcpServiceAccountKey);

    // Configure dataset and table names
    const datasetId = 'github_metrics';
    const tableId = 'issue_status_change';

    // Query to get the latest in_progress status
    const query = `
      SELECT date, repo, issue_number
      FROM \`${gcpServiceAccountKey.project_id}.${datasetId}.${tableId}\`
      WHERE repo = @repo
        AND issue_number = @issueNumber
        AND status = 'in_progress'
      ORDER BY date DESC
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

    if (rows.length === 0) {
      return null;
    }

    // Convert BigQueryTimestamp to string if needed
    const result = rows[0];
    if (result.date && result.date.value) {
      result.date = result.date.value;
    }

    return result;
  } catch (err) {
    sails.log.error('Error getting latest in_progress status from BigQuery:', err);
    return null;
  }
}

/**
 * Saves QA ready metrics to BigQuery
 *
 * @param {Object} data - The QA ready data to save
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 */
async function saveQaReadyToBigQuery(data, gcpServiceAccountKey) {
  try {
    // Get BigQuery client
    const bigquery = getBigQueryClient(gcpServiceAccountKey);

    // Configure dataset and table names
    const datasetId = 'github_metrics';
    const tableId = 'issue_qa_ready';

    // Get reference to the table
    const dataset = bigquery.dataset(datasetId);
    const table = dataset.table(tableId);

    // Insert the data
    await table.insert([data]);

    sails.log.debug('Successfully saved QA ready metrics to BigQuery:', {
      dataset: datasetId,
      table: tableId,
      data: data
    });

  } catch (err) {
    handleBigQueryError(err, 'saving QA ready to BigQuery', 'issue_qa_ready');
    throw err;
  }
}

/**
 * Saves release ready metrics to BigQuery
 *
 * @param {Object} data - The release ready data to save
 * @param {Object} gcpServiceAccountKey - The GCP service account key
 */
async function saveReleaseReadyToBigQuery(data, gcpServiceAccountKey) {
  try {
    // Get BigQuery client
    const bigquery = getBigQueryClient(gcpServiceAccountKey);

    // Configure dataset and table names
    const datasetId = 'github_metrics';
    const tableId = 'issue_release_ready';

    // Get reference to the table
    const dataset = bigquery.dataset(datasetId);
    const table = dataset.table(tableId);

    // Insert the data
    await table.insert([data]);

    sails.log.debug('Successfully saved release ready metrics to BigQuery:', {
      dataset: datasetId,
      table: tableId,
      data: data
    });

  } catch (err) {
    handleBigQueryError(err, 'saving release ready to BigQuery', 'issue_release_ready');
    throw err;
  }
}

/**
 * Calculates time difference excluding weekends if enabled.
 * This function is copied from https://github.com/fleetdm/fleet/blob/d20ddf33280464b1377aba8f755eb74df2f72724/.github/actions/eng-metrics/src/github-client.js#L512,
 * where it is thoroughly unit tested.
 *
 * @param {Date} startTime - The start time
 * @param {Date} endTime - The end time
 * @returns {number} Time difference in seconds
 */
function calculateTimeExcludingWeekends(startTime, endTime) {
  if (!EXCLUDE_WEEKENDS) {
    // If weekend exclusion is disabled, return simple time difference
    return Math.floor((endTime - startTime) / 1000);
  }

  // Use the provided weekend exclusion logic
  const startDay = startTime.getUTCDay();
  const endDay = endTime.getUTCDay();

  // Case: Both start time and end time are on the same weekend
  if (
    (startDay === 0 || startDay === 6) &&
    (endDay === 0 || endDay === 6) &&
    Math.floor(endTime / (24 * 60 * 60 * 1000)) -
      Math.floor(startTime / (24 * 60 * 60 * 1000)) <=
      2
  ) {
    // Return 0 seconds
    return 0;
  }

  // Make copies to avoid modifying original dates
  const adjustedStartTime = new Date(startTime);
  const adjustedEndTime = new Date(endTime);

  // Set to start of Monday if start time is on weekend
  if (startDay === 0) {
    // Sunday
    adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 1);
    adjustedStartTime.setUTCHours(0, 0, 0, 0);
  } else if (startDay === 6) {
    // Saturday
    adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 2);
    adjustedStartTime.setUTCHours(0, 0, 0, 0);
  }

  // Set to start of Saturday if end time is on Sunday
  if (endDay === 0) {
    // Sunday
    adjustedEndTime.setUTCDate(adjustedEndTime.getUTCDate() - 1);
    adjustedEndTime.setUTCHours(0, 0, 0, 0);
  } else if (endDay === 6) {
    // Saturday
    adjustedEndTime.setUTCHours(0, 0, 0, 0);
  }

  // Calculate raw time difference in milliseconds
  const weekendDays = countWeekendDays(adjustedStartTime, adjustedEndTime);
  const diffMs = adjustedEndTime - adjustedStartTime - weekendDays * 24 * 60 * 60 * 1000;

  // Ensure we don't return negative values
  return Math.max(0, Math.floor(diffMs / 1000));
}

/**
 * Counts the number of weekend days between two dates
 *
 * @param {Date} startDate - The start date
 * @param {Date} endDate - The end date
 * @returns {number} Number of weekend days
 */
function countWeekendDays(startDate, endDate) {
  // Make local copies of dates
  startDate = new Date(startDate);
  endDate = new Date(endDate);

  // Ensure startDate is before endDate
  if (startDate > endDate) {
    [startDate, endDate] = [endDate, startDate];
  }

  // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
  if (startDate.getUTCDay() === 0) {
    startDate.setUTCDate(startDate.getUTCDate() + 1);
  } else if (startDate.getUTCDay() === 6) {
    startDate.setUTCDate(startDate.getUTCDate() + 2);
  }
  if (endDate.getUTCDay() === 0) {
    endDate.setUTCDate(endDate.getUTCDate() - 2);
  } else if (endDate.getUTCDay() === 6) {
    endDate.setUTCDate(endDate.getUTCDate() - 1);
  }

  let count = 0;
  const current = new Date(startDate);

  while (current <= endDate) {
    const day = current.getUTCDay();
    if (day === 0 || day === 6) {
      // Sunday (0) or Saturday (6)
      count++;
    }
    current.setUTCDate(current.getUTCDate() + 1);
  }

  return count;
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
    handleBigQueryError(err, 'saving to BigQuery', 'issue_status_change');
    throw err;
  }
}
