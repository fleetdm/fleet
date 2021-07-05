module.exports = {


  friendlyName: 'View docs template',


  description: 'Display "Docs template" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/docs-template'
    }

  },


  fn: async function () {

    // Respond with view.
    return {
      outline: {
        sections: [
          {
            title: 'Get started',
            topics: [], // NOTE this array is empty bc this was not stubbed out
          },
          {
            title: 'Using Fleet',
            topics: [
              {
                title: 'Fleet UI',
                subtopics: ['Running queries', 'Scheduling queries'],
                relatedTopics: ['Osquery queries', 'Osquery packs'],
              },
              {
                title: 'fleetctl',
                subtopics: [], // NOTE this array is empty bc this was not stubbed out
                relatedTopics: [], // NOTE this array is empty bc this was not stubbed out
              },
              {
                title: 'REST API',
                subtopics: [], // NOTE this array is empty bc this was not stubbed out
                relatedTopics: [], // NOTE this array is empty bc this was not stubbed out
              },
            ],
          },
          {
            title: 'Adding endpoints',
            topics: [],
          },
          {
            title: 'Deployment',
            topics: [],
          },
          {
            title: 'Contributing',
            topics: [],
          },
        ],
      },
      currentPage: {
        section: 'Using Fleet',
        topic: 'Fleet UI',
      },
      body: [
        {
          type: 'subtopic',
          content: 'Running queries',
        },
        {
          type: 'text',
          content: 'The Fleet application allows you to query hosts that you have installed osquery on. To run a new query, navigate to "Queries" from the top nav, and then hit the "Create new query" button from the Queries page. From here, you can compose your query, view SQL table documentation via the sidebar, select arbitrary hosts (or groups of hosts), and execute your query. As results are returned, they will populate the interface in real time. You can use the integrated filtering tool to perform useful initial analytics and easily export the entire dataset for offline analysis.',
        },
        {
          type: 'image',
          content: '/images/fleetctl-900x580@2x.png',
          altText: 'An image of Fleet ctl'
        },
        {
          type: 'text',
          content: 'After you\'ve composed a query that returns the information you were looking for, you may choose to save the query. You can still continue to execute the query on whatever set of hosts you would like after you have saved the query.'
        },
        {
          type: 'note',
          content: 'To learn more about scheduling queries so that they run on an on-going basis, see the scheduling queries guide below.'
        },
        {
          type: 'subtopic',
          content: 'Scheduling queries',
        },
        {
          type: 'text',
          content: 'As discussed in the running queries documentation, you can use the Fleet application to create, execute, and save osquery queries. You can organize these queries into "Query Packs". To view all saved packs and perhaps create a new pack, select "Packs" from the top nav. Packs are usually organized by the general class of instrumentation that you\'re trying to perform.',
        },
        {
          type: 'text',
          content: 'To add queries to a pack, use the right-hand sidebar. You can take an existing scheduled query and add it to the pack. You must also define a few key details such as:',
        },
        {
          type: 'bullets',
          content: {
            intro: 'To add queries to a pack, use the right-hand sidebar. You can take an existing scheduled query and add it to the pack. You must also define a few key details such as:',
            bullets: [
              'interval: how often should the query be executed?',
              'logging: which osquery logging format would you like to use?',
              'platform: which operating system platforms should execute this query?',
              'minimum osquery version: if the table was introduced in a newer version of osquery, you may want to ensure that only sufficiently recent version of osquery execute the query.',
              'shard: from 0 to 100, what percent of hosts should execute this query?'
            ],
          }
        }
      ]
    };

  }

};
