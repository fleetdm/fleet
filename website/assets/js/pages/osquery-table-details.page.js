parasails.registerPage('osquery-table-details', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    selectedPlatform: 'all',
    search: '',
    showTableNav: false,
    userFriendlyPlatformNames: {
      'darwin': 'macOS',
      'linux': 'Linux',
      'windows': 'Windows',
      'chrome': 'ChromeOS',
      'all': 'All platforms'
    },
  },

  computed: {
    filteredTables: function () {
      return this.allTables.filter(
        (table) =>
          this._isIncluded(table.platforms, this.selectedPlatform)
      );
    },
    numberOfTablesDisplayed: function() {
      return this.filteredTables.length;
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {

    // Algolia DocSearch
    if(this.algoliaPublicKey) { // Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the documentation search will be disabled.
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search tables',
        debug: false,
        searchParameters: {
          'facetFilters': ['section:tables']
        },
      });
    }

    // Check the URL to see if a platformFilter was provided.
    if(window.location.search) {
      // https://caniuse.com/mdn-api_urlsearchparams_get
      let possibleSearchParamsToFilterBy = new URLSearchParams(window.location.search);
      let platformToFilterBy = possibleSearchParamsToFilterBy.get('platformFilter');
      // If the provided platform matches a key in the userFriendlyPlatformNames array, we'll set this.selectedPlatform.
      if(platformToFilterBy && this.userFriendlyPlatformNames[platformToFilterBy]){
        this.selectedPlatform = platformToFilterBy;
      }
    }

    // sort the array of all tables
    this.allTables = this.allTables.sort((a, b)=>{
      return a.title > b.title ? 1 : -1;
    });
    let keywordsForThisTable = [];
    if(this.tableToDisplay.keywordsForSyntaxHighlighting){
      keywordsForThisTable = this.tableToDisplay.keywordsForSyntaxHighlighting;
    }
    keywordsForThisTable = keywordsForThisTable.sort((a,b)=>{// Sorting the array of keywords by length to match larger keywords first.
      return a.length < b.length ? 1 : -1;
    });
    keywordsForThisTable = _.pull(keywordsForThisTable, this.tableToDisplay.title);
    (()=>{
      $('pre code').each((i, block) => {
        let tableNamesToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let match of block.innerHTML.match(this.tableToDisplay.title)||[]){
          tableNamesToHighlight.push(match);
        }
        // Now iterate through the keywordsToHighlight, replacing all matches in the elements innerHTML.
        let replacementHMTL = block.innerHTML;
        for(let keywordInExample of tableNamesToHighlight) {
          let regexForThisExample = new RegExp(keywordInExample, 'g');
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-attr">'+keywordInExample+'</span>');
        }
        // $(block).html(replacementHMTL);
        let columnNamesToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let keyword of keywordsForThisTable){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
          for(let match of block.innerHTML.match(keyword)||[]){
            columnNamesToHighlight.push(match);
          }
        }
        // Now iterate through the keywordsToHighlight, replacing all matches in the elements innerHTML.
        // let replacementHMTL = block.innerHTML;
        for(let keywordInExample of columnNamesToHighlight) {
          let regexForThisExample = new RegExp(keywordInExample, 'g');
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-string">'+keywordInExample+'</span>');
        }
        $(block).html(replacementHMTL);
        // After we've highlighted our keywords, we'll highlight the rest of the codeblock
        window.hljs.highlightElement(block);
        // If this example is a single-line, we'll do some basic formatting to make it more human-readable.
        if(!$(block)[0].innerText.match(/\n/gmi)){
          // Adding [purpose="line-break"] to SQL keywords if they are one of: SELECT, WHERE, FROM, JOIN. (case-insensitive)
          $('.hljs-keyword').each((i, el)=>{
            for(i in el.innerText.match(/select|where|from|join/gi)) {
              $(el).attr({'purpose':'line-break'});
            }
          });
        }
      });
    })();
    // Adjust the height of the sidebar navigation to match the height of the html partial
    (()=>{
      $('[purpose="table-of-contents"]').css({'max-height': 120});
      let tablePartialHeight = $('[purpose="table-container"]').height();
      $('[purpose="table-of-contents"]').css({'max-height': tablePartialHeight - 120});
    })();
  },
  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
    clickFilterByPlatform: async function(platform) {
      this.selectedPlatform = platform;
    },

    clickToggleTableNav: function() {
      this.showTableNav = !this.showTableNav;
    },

    _isIncluded: function (data, selectedOption) {
      if (selectedOption.startsWith('all') || selectedOption === '') {
        return true;
      }
      if (_.isArray(data)) {
        data = data.join(', ');
      }
      return (
        _.isString(data) && data.toLowerCase().includes(selectedOption.toLowerCase())
      );
    },
  }
});
