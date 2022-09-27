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
      if(a.name < b.name){
        return -1;
      }
    });
    let keywordsForThisTable = [];
    if(this.tableToDisplay.keywordsForSyntaxHighlighting){
      keywordsForThisTable = this.tableToDisplay.keywordsForSyntaxHighlighting;
    }
    (()=>{
      $('pre code').each((i, block) => {
        window.hljs.highlightBlock(block);
      });
      $('.hljs').each((i, el)=>{
        let keywordsInExample = _.filter(keywordsForThisTable, (word)=>{
          return _.includes(_.words(el.innerText, /[^, ]+/g), word);
        });
        for(let keyword of keywordsInExample) {
          let replacementHMTL = el.innerHTML.replaceAll(keyword, '<span class="hljs-attr">'+keyword+'</span>');
          $(el).html(replacementHMTL);
        }
      });
      $('.hljs-keyword').each((i, el)=>{
        for(i in el.innerText.match(/select|where|from|join/gi)) {
          $(el).addClass('line-break');
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
