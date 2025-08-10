module.exports = {

  friendlyName: 'Parse GCP service account key',

  description: 'Parses and validates a GCP service account key from configuration.',

  inputs: {
    gcpServiceAccountKey: {
      type: 'ref',
      description: 'The GCP service account key (can be a string or object)',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully parsed GCP service account key.',
      outputType: 'ref'
    },
    invalid: {
      description: 'The GCP service account key is invalid or missing required fields.'
    }
  },

  fn: async function ({ gcpServiceAccountKey }) {
    if (!gcpServiceAccountKey) {
      sails.log.error('No GCP service account key provided');
      throw 'invalid';
    }

    try {
      let parsedKey;

      // Check if it's already an object or needs parsing
      if (typeof gcpServiceAccountKey === 'object') {
        parsedKey = gcpServiceAccountKey;
      } else if (typeof gcpServiceAccountKey === 'string') {
        // Fix common JSON formatting issues before parsing
        let jsonString = gcpServiceAccountKey;

        // This handles cases where the private key has literal newlines
        jsonString = jsonString.replace(/"private_key":\s*"([^"]+)"/g, (match, key) => {
          // Replace actual newlines with escaped newlines only within the private key value
          const fixedKey = key.replace(/\n/g, '\\n');
          return `"private_key": "${fixedKey}"`;
        });

        // Parse the cleaned JSON
        parsedKey = JSON.parse(jsonString);
      } else {
        throw new Error('Invalid GCP service account key type');
      }

      // Validate that it has the expected structure
      if (!parsedKey.type || !parsedKey.project_id || !parsedKey.private_key) {
        throw new Error('Invalid GCP service account key structure');
      }

      return parsedKey;
    } catch (err) {
      sails.log.error('Failed to parse GCP service account key:', err);
      throw 'invalid';
    }
  }

};
