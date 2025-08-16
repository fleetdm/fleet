module.exports = {

  friendlyName: 'Get BigQuery client',

  description: 'Creates a new BigQuery client instance.',

  inputs: {
    gcpServiceAccountKey: {
      type: 'ref',
      description: 'The GCP service account key',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully created BigQuery client.',
      outputType: 'ref'
    }
  },

  fn: async function ({ gcpServiceAccountKey }) {
    const {BigQuery} = require('@google-cloud/bigquery');

    return new BigQuery({
      projectId: gcpServiceAccountKey.project_id,
      credentials: gcpServiceAccountKey
    });
  }

};
