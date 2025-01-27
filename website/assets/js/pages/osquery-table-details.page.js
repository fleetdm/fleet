parasails.registerPage('osquery-table-details', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    selectedPlatform: 'apple',
    search: '',
    showTableNav: false,
    userFriendlyPlatformNames: {
      'darwin': 'macOS',
      'linux': 'Linux',
      'windows': 'Windows',
      'chrome': 'ChromeOS',
      'all': 'All platforms'
    },
    modal: undefined,
  },

  computed: {
    filteredTables: function () {
      let platformToFilterBy = this.selectedPlatform !== 'apple' ? this.selectedPlatform : 'darwin';
      return this.allTables.filter(
        (table) =>
          table.platforms.includes(platformToFilterBy)
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
    if(['#apple','#linux','#windows','#chrome'].includes(window.location.hash)){
      this.selectedPlatform = window.location.hash.split('#')[1];
    } else {
      // otherwise, default the filter to be the first supported platform of the currently viewed table.
      this.selectedPlatform = this.tableToDisplay.platforms[0] === 'darwin' ? 'apple' : this.tableToDisplay.platforms[0];
    }
  },
  mounted: async function() {

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
    // 5 ms after the page loads, scroll the table of contents to the currently active link.
    await setTimeout(()=>{
      let activeTableLink = $('[purpose="table-of-contents-link"].active')[0];
      if(activeTableLink) {
        $('[purpose="table-of-contents"]')[0].scrollTop = activeTableLink.offsetTop - 12;
      }
      // Note: we're running this code after a 5ms delay to make sure the tables have been filtered, otherwise it will scroll the table of contents to the links posiiton in the full list of tables.
    }, 5);
  },
  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
    clickFilterByPlatform: async function(platform) {
      this.selectedPlatform = platform;
    },
    clickSelectPlatform: function (platform) {
      let platformToLookFor = platform;
      if(platform === 'apple'){
        platformToLookFor = 'darwin';
      }
      let currentTableAvailableOnNewPlatform = this.tableToDisplay.platforms.includes(platformToLookFor);
      if(!currentTableAvailableOnNewPlatform){
        if(platformToLookFor === 'chrome'){
          this.goto('/tables/chrome_extensions#chrome');
        } else if(platformToLookFor === 'darwin') {
          this.goto('/tables/account_policy_data#apple');
        } else if(platformToLookFor === 'linux') {
          this.goto('/tables/apparmor_events#linux');
        } else if(platformToLookFor === 'windows') {
          this.goto('/tables/appcompat_shims#windows');
        }
      } else {
        this.selectedPlatform = platform;
      }

    },
    clickToggleTableNav: function() {
      this.showTableNav = !this.showTableNav;
    },
    clickOpenTablesNav: function() {
      this.modal = 'table-of-contents';
    },
    closeModal: async function() {
      this.modal = '';
      await this.forceRender();
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
