/**
 * <docs-nav-and-search>
 * -----------------------------------------------------------------------------
 * A navigation bar with a search box.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('docsNavAndSearch', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'searchFilter',
    'algoliaPublicKey',
    'currentSection',
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      //…
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="d-block">
    <div class="d-flex flex-row w-100 justify-content-between">
      <div purpose="nav-link-container" class="d-flex align-items-center">
        <div purpose="docs-links" class="d-flex flex-row">
          <a :class="[currentSection === 'docs' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/docs" style="text-decoration: none; text-decoration-line: none;">Get started</a>
          <a :class="[currentSection === 'software' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/software-catalog" style="text-decoration: none; text-decoration-line: none;">Apps</a>
          <a :class="[currentSection === 'controls' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/mdm-commands" style="text-decoration: none; text-decoration-line: none;">Controls</a>
          <a :class="[currentSection === 'vitals' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/vitals" style="text-decoration: none; text-decoration-line: none;">Vitals</a>
          <a :class="[currentSection === 'reports' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/reports" style="text-decoration: none; text-decoration-line: none;">Reports</a>
          <a :class="[currentSection === 'policies' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/policies" style="text-decoration: none; text-decoration-line: none;">Policies</a>
          <a :class="[currentSection === 'tables' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/tables" style="text-decoration: none; text-decoration-line: none;">Data tables</a>
        </div>
      </div>
      <div>
        <div purpose="nav-bar-search" id="docsearch-query" class="d-flex" v-if="searchFilter === 'tables' && algoliaPublicKey">
          <div purpose="disabled-search" class="d-flex">
            <div class="input-group d-flex flex-nowrap">
              <div class="input-group-prepend">
                <span class="input-group-text border-0 bg-transparent" >
                  <img style="height: 16px; width: 16px;" class="search" alt="search" src="/images/icon-search-16x16@2x.png">
                </span>
              </div>
              <div class="form-control border-0 ">
              <input class="docsearch-input pr-1"
                placeholder="Search" aria-label="Search"
                />
              </div>
            </div>
          </div>
        </div>
        <div purpose="nav-bar-search" class="d-flex" v-else>
          <div purpose="searchbar" class="d-flex">
            <div class="input-group d-flex flex-nowrap">
              <div class="input-group-prepend">
                <span class="input-group-text border-0 bg-transparent pr-0" >
                  <img style="height: 16px; width: 16px;" class="search" alt="search" src="/images/icon-search-16x16@2x.png">
                </span>
              </div>
              <form purpose="google-search" id="docs-nav-search-form">
                <div class="form-control border-0">
                  <input id="nav-search-bar" placeholder="Search" aria-label="Search"/>
                </div>
              </form>
              <button type="submit" form="docs-nav-search-form" aria-label="Search" class="input-group-append d-flex align-items-center" purpose="searchbar-submit">⏎</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    // Note: Only the data tables pages use Algolia DocSearch. All other pages send search queries to Google.
    if(this.searchFilter === 'tables' && this.algoliaPublicKey) {
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search data tables',
        debug: false,
        searchParameters: {
          'facetFilters': ['section:tables']
        },
        translations: {
          button: {
            buttonText: 'Search data tables',
            buttonAriaLabel: 'Search data tables',
          },
        },
      });
    }
  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  },
});
