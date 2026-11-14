parasails.registerPage('integrations', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedCategory: 'all',
    searchQuery: '',
    scrollDistance: 0,
    lastScrollTop: 0,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    window.addEventListener('scroll', this.handleScrollingCategoryNav);
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

    countVisibleIntegrations: function() {
      return this.integrations.filter((integration)=> this.isIntegrationVisible(integration.slug)).length;
    },

    clickClearFilters: function() {
      this.selectedCategory = 'all';
      this.searchQuery = '';
    },

    // Uses the same thresholds as the layout's header, so the sticky nav moves up when the header hides.
    handleScrollingCategoryNav: function() {
      let categoryNav = document.querySelector('[purpose="category-nav"]');
      let scrollTop = window.pageYOffset;
      if (scrollTop > this.scrollDistance && scrollTop > window.innerHeight * 1.5) {
        categoryNav.classList.add('header-hidden');
        this.lastScrollTop = scrollTop;
      } else if (scrollTop < this.lastScrollTop - 60) {
        categoryNav.classList.remove('header-hidden');
        this.lastScrollTop = scrollTop;
      }
      this.scrollDistance = scrollTop;
    },

  }
});
