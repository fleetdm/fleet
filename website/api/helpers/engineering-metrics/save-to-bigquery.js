module.exports = {

  friendlyName: 'Save to BigQuery',

  description: 'Saves data to a specified BigQuery table.',

  inputs: {
    data: {
      type: 'ref',
      description: 'The data to save',
      required: true
    },
    gcpServiceAccountKey: {
      type: 'ref',
      description: 'The GCP service account key',
      required: true
    },
    tableId: {
      type: 'string',
      description: 'The table name to save to',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully saved data to BigQuery.'
    }
  },

  fn: async function ({ data, gcpServiceAccountKey, tableId }) {
    try {
      // Get BigQuery client
      const {BigQuery} = require('@google-cloud/bigquery');

      const bigquery = new BigQuery({
        projectId: gcpServiceAccountKey.project_id,
        credentials: gcpServiceAccountKey
      });

      // Configure dataset and table names
      const datasetId = 'github_metrics';

      // Get reference to the table
      const dataset = bigquery.dataset(datasetId);
      const table = dataset.table(tableId);

      // Insert the data
      await table.insert([data]);

      sails.log.verbose(`Successfully saved data to BigQuery ${tableId}:`, {
        dataset: datasetId,
        table: tableId,
        data: data
      });

    } catch (err) {
      // Determine operation name based on table
      let operation = 'saving to BigQuery';
      if (tableId === 'issue_qa_ready') {
        operation = 'saving QA ready to BigQuery';
      } else if (tableId === 'issue_release_ready') {
        operation = 'saving release ready to BigQuery';
      } else if (tableId === 'issue_status_change') {
        operation = 'saving status change to BigQuery';
      }

      // Handle specific BigQuery errors
      if (err.name === 'PartialFailureError') {
        // Log the specific rows that failed
        throw new Error(`Partial failure when ${operation}:`, err.errors);
      } else if (err.code === 404) {
        throw new Error('BigQuery table or dataset not found. Please ensure the table exists:', {
          dataset: 'github_metrics',
          table: tableId,
          fullError: err.message
        });
      } else if (err.code === 403) {
        throw new Error('Permission denied when accessing BigQuery. Check service account permissions.');
      }
      throw new Error(`Error ${operation}:`, err);
    }
  }

};
