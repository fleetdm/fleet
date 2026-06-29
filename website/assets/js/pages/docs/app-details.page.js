parasails.registerPage('app-details', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {

    let columnNamesForThisQuery = ['bundle_short_version', 'bundle_identifier', 'version', 'name'];
    let tableNamesForThisQuery = ['programs', 'apps'];
    (()=>{
      $('pre code').each((i, block) => {
        if(block.classList.contains('ps') || block.classList.contains('sh')){
          window.hljs.highlightElement(block);
          return;
        } else {
          let tableNamesToHighlight = [];// Empty array to track the keywords that we will need to highlight
          for(let tableName of tableNamesForThisQuery){// Going through the array of keywords for this table, if the entire word matches, we'll add it to the
            for(let match of block.innerHTML.match(tableName)|| []){
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
    //…
  }
});
