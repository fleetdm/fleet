parasails.registerPage('vital-details', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    contributors: [],
    selectedPlatform: 'macos', // Initially set to 'macos'
    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function () {
    //…
  },
  mounted: async function () {
    // Set the selected platform from the user's hash.
    // This lets us persist the user's filter when they navigate to a new page.
    if(['#macos','#linux','#windows','#chrome'].includes(window.location.hash)){
      this.selectedPlatform = window.location.hash.split('#')[1];
    }

    // if(this.algoliaPublicKey) { // Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the documentation search will be disabled.
    //   docsearch({
    //     appId: 'NZXAYZXDGH',
    //     apiKey: this.algoliaPublicKey,
    //     indexName: 'fleetdm',
    //     container: '#docsearch-query',
    //     placeholder: 'Search',
    //     debug: false,
    //     searchParameters: {
    //       'facetFilters': ['section:queries']
    //     },
    //   });
    // }
    let columnNamesForThisQuery = [];
    let tableNamesForThisQuery = [];
    if(this.columnNamesForSyntaxHighlighting){
      columnNamesForThisQuery = this.columnNamesForSyntaxHighlighting;
    }
    if(this.tableNamesForSyntaxHighlighting){
      tableNamesForThisQuery = this.tableNamesForSyntaxHighlighting;
    }
    // Sorting the arrays of keywords by length to match larger keywords first.
    columnNamesForThisQuery = columnNamesForThisQuery.sort((a,b)=>{
      return a.length < b.length ? 1 : -1;
    });
    tableNamesForThisQuery = tableNamesForThisQuery.sort((a,b)=>{
      return a.length < b.length ? 1 : -1;
    });
    (()=>{
      $('pre code').each((i, block) => {
        let tableNamesToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let tableName of tableNamesForThisQuery){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
          for(let match of block.innerHTML.match(tableName)||[]){
            tableNamesToHighlight.push(match);
          }
        }
        // Now iterate through the tableNamesToHighlight, replacing all matches in the elements innerHTML.
        let replacementHMTL = block.innerHTML;
        for(let keywordInExample of tableNamesToHighlight) {
          let regexForThisExample = new RegExp(keywordInExample, 'g');
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-attr">'+_.trim(keywordInExample)+'</span> ');
        }
        $(block).html(replacementHMTL);
        let columnNamesToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let columnName of columnNamesForThisQuery){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
          for(let match of block.innerHTML.match(columnName)||[]){
            columnNamesToHighlight.push(match);
          }
        }

        for(let keywordInExample of columnNamesToHighlight) {
          let regexForThisExample = new RegExp(keywordInExample, 'g');
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-string">'+_.trim(keywordInExample)+'</span>');
        }
        $(block).html(replacementHMTL);
        window.hljs.highlightElement(block);
        // After we've highlighted our keywords, we'll highlight the rest of the codeblock
        // If this example is a single-line, we'll do some basic formatting to make it more human-readable.
        if($(block)[0].innerText.match(/\n/gmi)){
          $(block).addClass('has-linebreaks');
        } else {
          $(block).addClass('no-linebreaks');
        }
      });
    })();
    $('[purpose="copy-button"]').on('click', async function() {
      let code = $(this).siblings('pre').find('code').text();
      $(this).addClass('copied');
      await setTimeout(()=>{
        $(this).removeClass('copied');
      }, 2000);
      navigator.clipboard.writeText(code);
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickSelectPlatform: function (platform) {
      this.selectedPlatform = platform;
    },
    clickOpenTableOfContents: function () {
      this.modal = 'table-of-contents';
    },
    closeModal: async function() {
      this.modal = '';
      await this.forceRender();
    }
  },
});
