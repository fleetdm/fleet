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
      max: 10,// The search API only serves the first 100 results (10 pages of 10).
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

    // If this Sails app is not configured with Google Programmable Search credentials,
    // fall back to opening a scoped Google search (the pre-2026-09-30 behavior).
    if(!sails.config.custom.googleSearchApiKey || !sails.config.custom.googleSearchEngineId) {
      throw {redirect: 'https://www.google.com/search?q='+encodeURIComponent('site:fleetdm.com '+searchQuery)};
    }

    let searchResults = [];
    let formattedTotalResults = '0';
    let hasMoreResults = false;
    let searchFailed = false;

    if(searchQuery) {
      try {
        let apiResponse = await sails.helpers.http.get('https://www.googleapis.com/customsearch/v1', {
          key: sails.config.custom.googleSearchApiKey,
          cx: sails.config.custom.googleSearchEngineId,
          q: searchQuery,
          num: 10,
          start: (page - 1) * 10 + 1,
        });
        formattedTotalResults = apiResponse.searchInformation ? apiResponse.searchInformation.formattedTotalResults : '0';
        hasMoreResults = !!(apiResponse.queries && apiResponse.queries.nextPage) && page < 10;
        for(let item of (apiResponse.items || [])) {
          let displayPath = item.link;
          try {
            let parsedUrl = new URL(item.link);
            displayPath = parsedUrl.hostname + decodeURIComponent(parsedUrl.pathname).replace(/\/$/, '');
          } catch(unusedErr) { /* If the URL can't be parsed, display it as-is. */ }
          searchResults.push({
            url: item.link,
            title: item.title,
            snippet: (item.snippet || '').replace(/\s+/g, ' ').trim(),
            displayPath,
          });
        }
      } catch(err) {
        // Show a fallback link rather than an error page if the search API is unavailable (e.g. over quota).
        sails.log.warn('The Google search API returned an error when searching for "'+searchQuery+'":', err);
        searchFailed = true;
      }
    }

    // Respond with view.
    return {
      searchQuery,
      searchResults,
      formattedTotalResults,
      currentPage: page,
      hasMoreResults,
      searchFailed,
    };

  }


};
