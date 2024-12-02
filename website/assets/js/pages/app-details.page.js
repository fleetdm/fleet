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
