parasails.registerPage('integrations', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedCategory: 'all',
    searchQuery: '',
    visitorPlatform: 'macos',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Set before the AI demo mounts, so its animation starts with the visitor's own OS.
    // Everyone else (macOS, iOS, Android, ChromeOS) sees the macOS example.
    if (bowser.windows) {
      this.visitorPlatform = 'windows';
    } else if (bowser.linux) {
      this.visitorPlatform = 'linux';
    }
    // e.g., /integrations#linux previews another platform's example.
    if (['#macos', '#windows', '#linux'].includes(window.location.hash)) {
      this.visitorPlatform = window.location.hash.slice(1);
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickSelectCategory: function(categorySlug) {
      this.selectedCategory = categorySlug;
    },

    doesIntegrationMatchSearch: function(integrationSlug) {
      let query = this.searchQuery.trim().toLowerCase();
      if (!query) {
        return true;
      }
      let integration = _.find(this.integrations, {slug: integrationSlug});
      let category = _.find(this.integrationCategories, {slug: integration.category});
      return [integration.name, integration.summary, category.name].join(' ').toLowerCase().includes(query);
    },

    isIntegrationVisible: function(integrationSlug) {
      let integration = _.find(this.integrations, {slug: integrationSlug});
      if (this.selectedCategory !== 'all' && integration.category !== this.selectedCategory) {
        return false;
      }
      return this.doesIntegrationMatchSearch(integrationSlug);
    },

    countVisibleIntegrations: function(categorySlug) {
      return this.integrations.filter((integration)=>{
        return (!categorySlug || integration.category === categorySlug) && this.isIntegrationVisible(integration.slug);
      }).length;
    },

    clickClearFilters: function() {
      this.selectedCategory = 'all';
      this.searchQuery = '';
    },

  }
});
