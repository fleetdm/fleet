parasails.registerPage('query-detail', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    contributors: [],
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function () {
    //…
  },
  mounted: async function () {

    if(this.algoliaPublicKey) { // Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the documentation search will be disabled.
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search',
        debug: false,
        searchParameters: {
          'facetFilters': ['section:queries']
        },
      });
    }
    let keywordsForThisTable = [];
    if(this.keywordsForSyntaxHighlighting){
      keywordsForThisTable = this.keywordsForSyntaxHighlighting;
    }
    keywordsForThisTable = keywordsForThisTable.sort((a,b)=>{// Sorting the array of keywords by length to match larger keywords first.
      return a.length < b.length ? 1 : -1;
    });
    (()=>{
      $('pre code').each((i, block) => {
        let keywordsToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let keyword of keywordsForThisTable){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
          for(let match of block.innerHTML.match(keyword+' ')||[]){
            keywordsToHighlight.push(match);
          }
        }
        // Now iterate through the keywordsToHighlight, replacing all matches in the elements innerHTML.
        let replacementHMTL = block.innerHTML;
        for(let keywordInExample of keywordsToHighlight) {
          let regexForThisExample = new RegExp(keywordInExample, 'g');
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-attr">'+_.trim(keywordInExample)+'</span> ');
        }
        $(block).html(replacementHMTL);
        window.hljs.highlightElement(block);
        // After we've highlighted our keywords, we'll highlight the rest of the codeblock
        // If this example is a single-line, we'll do some basic formatting to make it more human-readable.
        if(!$(block)[0].innerText.match(/\n/gmi)){
          // Adding [purpose="line-break"] to SQL keywords if they are one of: SELECT, WHERE, FROM, JOIN, AND, or OR. (case-insensitive)
          $('.hljs-keyword').each((i, el)=>{
            for(i in el.innerText.match(/select|where|from|join|and|or/gi)) {
              $(el).attr({'purpose':'line-break'});
            }
          });
        }
      });
    })();
    // $('pre code').each((i, block) => {
    //   window.hljs.highlightElement(block);
    // });
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
    //…
  },
});
