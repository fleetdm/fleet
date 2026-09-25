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

    redirect: {
      responseType: 'redirect'
    },

  },


  fn: async function ({q, page}) {

    let searchQuery = q.trim();

    // If this Sails app is not configured with Vertex AI Search credentials, fall back
    // to opening a scoped Google search (the pre-2026-09-30 behavior).
    if(!sails.config.custom.googleSearchServingConfig || !sails.config.custom.googleSearchGcpServiceAccountKey) {
      throw {redirect: 'https://www.google.com/search?q='+encodeURIComponent('site:fleetdm.com '+searchQuery)};
    }

    let searchResults = [];
    let totalResults = 0;
    let hasMoreResults = false;
    let searchFailed = false;

    if(searchQuery) {
      try {
        let {google} = require('googleapis');
        let auth = new google.auth.GoogleAuth({
          credentials: sails.config.custom.googleSearchGcpServiceAccountKey,
          scopes: ['https://www.googleapis.com/auth/cloud-platform'],
        });
        let discoveryengine = google.discoveryengine({version: 'v1', auth});
        let apiResponse = await discoveryengine.projects.locations.collections.dataStores.servingConfigs.search({
          servingConfig: sails.config.custom.googleSearchServingConfig,
          requestBody: {
            query: searchQuery,
            pageSize: 10,
            offset: (page - 1) * 10,
            contentSearchSpec: {snippetSpec: {returnSnippet: true}},
          },
        }, {timeout: 10000});
        totalResults = apiResponse.data.totalSize || 0;
        hasMoreResults = page * 10 < totalResults && page < 10;
        for(let result of (apiResponse.data.results || [])) {
          let doc = result.document && result.document.derivedStructData;
          if(!doc || !doc.link) {
            continue;
          }
          // Only link to results with well-formed http(s) URLs.
          let displayPath;
          try {
            let parsedUrl = new URL(doc.link);
            if(!['http:', 'https:'].includes(parsedUrl.protocol)) {
              continue;
            }
            displayPath = parsedUrl.hostname + decodeURIComponent(parsedUrl.pathname).replace(/\/$/, '');
          } catch(unusedErr) {
            continue;
          }
          let snippet = '';
          if(_.isArray(doc.snippets) && doc.snippets[0] && doc.snippets[0].snippetStatus === 'SUCCESS') {
            // Snippets come back as HTML with <b> highlights; display them as plain text.
            // (Stripping repeats until stable so malformed nested tags can't survive a single pass.)
            let strippedSnippet = doc.snippets[0].snippet;
            let beforeStripping;
            do {
              beforeStripping = strippedSnippet;
              strippedSnippet = strippedSnippet.replace(/<[^>]*>?/g, '');
            } while(strippedSnippet !== beforeStripping);
            snippet = _.unescape(strippedSnippet).replace(/\s+/g, ' ').trim();
          } else if(_.isObject(doc.pagemap) && _.isArray(doc.pagemap.metatags) && doc.pagemap.metatags[0]) {
            // Basic website search doesn't generate snippets, so fall back to the page's meta description.
            snippet = doc.pagemap.metatags[0]['og:description'] || doc.pagemap.metatags[0]['description'] || '';
          }
          searchResults.push({
            url: doc.link,
            title: doc.title || displayPath,
            snippet,
            displayPath,
          });
        }
      } catch(err) {
        // Show a fallback link rather than an error page if the search API is unavailable (e.g. over quota).
        sails.log.warn('The Vertex AI Search API returned an error when searching for "'+searchQuery+'":', err);
        searchFailed = true;
      }
    }

    // Respond with view.
    return {
      searchQuery,
      searchResults,
      formattedTotalResults: totalResults.toLocaleString('en-US'),
      currentPage: page,
      hasMoreResults,
      searchFailed,
    };

  }


};
