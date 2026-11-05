module.exports = {


  friendlyName: 'View search',


  description: 'Display "Search" page: search results for fleetdm.com, scoped to this website.',


  inputs: {

    q: {
      type: 'string',
      description: 'The user\'s search query.',
      defaultsTo: '',
    },

    page: {
      type: 'number',
      description: 'Which page of search results to display.',
      defaultsTo: 1,
      min: 1,
      max: 10,
    },

  },


  exits: {

    success: {
      viewTemplatePath: 'pages/search'
    },

  },


  fn: async function ({q, page}) {

    let {google} = require('googleapis');

    if(!sails.config.custom.googleSearchGcpServiceAccountKey) {
      throw new Error('Cannot search: sails.config.custom.googleSearchGcpServiceAccountKey is not set. Set it to the parsed JSON key of a service account in the "fleetdm-website" GCP project.');
    }

    let searchQuery = q.trim();
    let searchResults = [];
    let totalResults = 0;

    if(searchQuery) {
      let {data} = await google.discoveryengine({
        version: 'v1',
        auth: new google.auth.GoogleAuth({
          credentials: sails.config.custom.googleSearchGcpServiceAccountKey,
          scopes: ['https://www.googleapis.com/auth/cloud-platform'],
        })
      }).projects.locations.collections.dataStores.servingConfigs.search({
        servingConfig: 'projects/fleetdm-website/locations/global/collections/default_collection/dataStores/DATA_STORE_ID/servingConfigs/default_search',
        requestBody: {query: searchQuery, pageSize: 10, offset: (page - 1) * 10},
      });
      totalResults = data.totalSize || 0;
      for(let result of (data.results || [])) {
        let doc = result.document.derivedStructData;
        // Only link to http(s) URLs, so a bad result can't become a javascript: link.
        if(!/^https?:\/\//.test(doc.link)) {
          continue;
        }//•
        let metaTagsByName = (doc.pagemap && doc.pagemap.metatags && doc.pagemap.metatags[0]) || {};
        searchResults.push({
          url: doc.link,
          title: doc.title || doc.link,
          snippet: metaTagsByName['og:description'] || metaTagsByName['description'] || '',
          displayPath: doc.link.replace(/^https?:\/\//, '').replace(/\/$/, ''),
        });
      }//∞
    }//ﬁ

    return {
      searchQuery,
      searchResults,
      formattedTotalResults: totalResults.toLocaleString('en-US'),
      currentPage: page,
      hasMoreResults: page < 10 && page * 10 < totalResults,
    };

  }


};
