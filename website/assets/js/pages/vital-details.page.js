parasails.registerPage('vital-details', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    contributors: [],
    selectedPlatform: 'apple', // Initially set to 'macos'
    selectedTab: 'sql',
    modal: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function () {
    //…
  },
  mounted: async function () {
    // Set the selected platform from the hash in the user's URL.
    // All links to vitals in the on-page navigation have the currently selected filter appended to them, this lets us persist the user's filter when they navigate to a new page.
    if(['#apple','#linux','#windows','#chrome'].includes(window.location.hash)){
      this.selectedPlatform = window.location.hash.split('#')[1];
    } else if(bowser.windows){
      this.selectedPlatform = 'windows';
    }
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
        if(block.classList.contains('ps') || block.classList.contains('sh')){
          window.hljs.highlightElement(block);
          return;
        }
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
          replacementHMTL = replacementHMTL.replace(regexForThisExample, '<span class="hljs-attr">'+_.trim(keywordInExample)+'</span>');
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
      let code = $(this).closest('[purpose="codeblock"]').find('pre:visible code').text();
      $(this).addClass('copied');
      await setTimeout(()=>{
        $(this).removeClass('copied');
      }, 2000);
      navigator.clipboard.writeText(code);
    });
    window.addEventListener('scroll', this.handleScrollingPlatformFilters);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickSelectPlatform: function (platform) {
      let platformToLookFor = platform;
      if(platform === 'apple'){
        platformToLookFor = 'darwin';
      }
      let currentVitalAvailableOnNewPlatform = this.thisVital.platform.includes(platformToLookFor);

      if(!currentVitalAvailableOnNewPlatform){
        if(platformToLookFor === 'chrome'){
          this.goto('/vitals/'+this.chromeVitals[0].slug+'#chrome');
        } else if(platformToLookFor === 'darwin') {
          this.goto('/vitals/'+this.macOsVitals[0].slug+'#apple');
        } else if(platformToLookFor === 'linux') {
          this.goto('/vitals/'+this.linuxVitals[0].slug+'#linux');
        } else if(platformToLookFor === 'windows') {
          this.goto('/vitals/'+this.windowsVitals[0].slug+'#windows');
        }
      } else {
        this.selectedPlatform = platform;

      }

    },
    handleScrollingPlatformFilters: function () {
      let platformFilters = document.querySelector('div[purpose="platform-filters"]');
      let scrollTop = window.pageYOffset;
      let windowHeight = window.innerHeight;
      // If the right nav bar exists, add and remove a class based on the current scroll position.
      if (platformFilters) {
        if (scrollTop > this.scrollDistance && scrollTop > windowHeight * 1.5) {
          platformFilters.classList.add('header-hidden');
          this.lastScrollTop = scrollTop;
        } else if(scrollTop < this.lastScrollTop - 60) {
          platformFilters.classList.remove('header-hidden');
          this.lastScrollTop = scrollTop;
        }
      }
      this.scrollDistance = scrollTop;
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
