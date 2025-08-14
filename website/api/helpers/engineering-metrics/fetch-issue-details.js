module.exports = {

  friendlyName: 'Fetch issue details',

  description: 'Fetches issue details from GitHub API using the node ID.',

  inputs: {
    nodeId: {
      type: 'string',
      description: 'The GitHub node ID of the issue',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully fetched issue details.',
      outputType: 'ref'
    }
  },

  fn: async function ({ nodeId }) {
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
  }

};
