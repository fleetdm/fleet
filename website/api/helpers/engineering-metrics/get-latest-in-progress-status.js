module.exports = {

  friendlyName: 'Get latest in progress status',

  description: 'Gets the latest in_progress status entry from BigQuery for an issue.',

  inputs: {
    repo: {
      type: 'string',
      description: 'The repository name (e.g., "fleetdm/fleet")',
      required: true
    },
    issueNumber: {
      type: 'number',
      description: 'The issue number',
      required: true
    },
    gcpServiceAccountKey: {
      type: 'ref',
      description: 'The GCP service account key',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully retrieved latest in_progress status.',
      outputType: 'ref'
    }
  },

  fn: async function ({ repo, issueNumber, gcpServiceAccountKey }) {
    try {
      // Get BigQuery client
      const bigquery = await sails.helpers.engineeringMetrics.getBigqueryClient.with({
        gcpServiceAccountKey: gcpServiceAccountKey
      });

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
      // Handle specific BigQuery errors
      if (err.name === 'PartialFailureError') {
        // Log the specific rows that failed
        sails.log.error('Partial failure when getting latest in_progress status from BigQuery:', err.errors);
      } else if (err.code === 404) {
        sails.log.error('BigQuery table or dataset not found. Please ensure the table exists:', {
          dataset: 'github_metrics',
          table: 'issue_status_change',
          fullError: err.message
        });
      } else if (err.code === 403) {
        sails.log.error('Permission denied when accessing BigQuery. Check service account permissions.');
      } else {
        sails.log.error('Error getting latest in_progress status from BigQuery:', err);
      }
      return null;
    }
  }

};
