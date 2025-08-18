module.exports = {

  friendlyName: 'Check if record exists',

  description: 'Checks if a record exists in a BigQuery table.',

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
    },
    tableId: {
      type: 'string',
      description: 'The table name',
      required: true
    },
    additionalCondition: {
      type: 'string',
      description: 'Optional additional WHERE clause condition',
      defaultsTo: ''
    }
  },

  exits: {
    success: {
      description: 'Successfully checked if record exists.',
      outputType: 'boolean'
    }
  },

  fn: async function ({ repo, issueNumber, gcpServiceAccountKey, tableId, additionalCondition }) {
    try {
      // Get BigQuery client
      const {BigQuery} = require('@google-cloud/bigquery');

      const bigquery = new BigQuery({
        projectId: gcpServiceAccountKey.project_id,
        credentials: gcpServiceAccountKey
      });
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
      // Handle specific BigQuery errors
      if (err.name === 'PartialFailureError') {
        // Log the specific rows that failed
        sails.log.error(`Partial failure when checking if record exists in ${tableId}:`, err.errors);
      } else if (err.code === 404) {
        sails.log.error('BigQuery table or dataset not found. Please ensure the table exists:', {
          dataset: 'github_metrics',
          table: tableId,
          fullError: err.message
        });
        return false;
      } else if (err.code === 403) {
        sails.log.error('Permission denied when accessing BigQuery. Check service account permissions.');
      } else {
        sails.log.error(`Error checking if record exists in ${tableId}:`, err);
      }
      return false;
    }
  }

};
