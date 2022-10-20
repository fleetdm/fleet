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
      'all': 'All platforms'
    },
  },

  computed: {
    filteredTables: function () {
      return this.allTables.filter(
        (table) =>
          this._isIncluded(table.platforms, this.selectedPlatform) &&
          this._isIncluded(table.title, this.search)
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
    // sort the array of all tables
    this.allTables = this.allTables.sort((a, b)=>{
      if(a.title < b.title){
        return -1;
      }
    });
    let keywordsForThisTable = [];
    if(this.tableToDisplay.keywordsForSyntaxHighlighting){
      keywordsForThisTable = this.tableToDisplay.keywordsForSyntaxHighlighting;
    }
    keywordsForThisTable = keywordsForThisTable.sort((a,b)=>{// Sorting the array of keywords by length to match larger keywords first.
      if(a.length > b.length){
        return -1;
      }
    });
    (()=>{
      $('pre code').each((i, block) => {
        let keywordsToHighlight = [];// Empty array to track the keywords that we will need to highlight
        for(let keyword of keywordsForThisTable){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
          for(let match of block.innerHTML.match(keyword)||[]){
            keywordsToHighlight.push(match);
          }
        }
        // Now iterate through the keywordsToHighlight, replacing all matches in the elements innerHTML.
        let replacementHMTL = block.innerHTML;
        for(let keywordInExample of keywordsToHighlight) {
          replacementHMTL = replacementHMTL.replaceAll(keywordInExample, '<span class="hljs-attr">'+keywordInExample+'</span>');
        }
        $(block).html(replacementHMTL);
        // After we've highlighted our keywords, we'll highlight the rest of the codeblock
        window.hljs.highlightBlock(block);
      });
      // Adding [purpose="line-break"] to SQL keywords if they are one of: SELECT, WHERE, FROM, JOIN. (case-insensitive)
      $('.hljs-keyword').each((i, el)=>{
        for(i in el.innerText.match(/select|where|from|join/gi)) {
          $(el).attr({'purpose':'line-break'});
        }
      });
    })();
    // Adjust the height of the sidebar navigation to match the height of the html partial
    (()=>{
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
