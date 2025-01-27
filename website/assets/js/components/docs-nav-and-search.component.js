/**
 * <docs-nav-and-search>
 * -----------------------------------------------------------------------------
 * A navigation bar with a search box.
 *
 * @type {Component}
 *
 * @event click   [emitted when clicked]
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
          <a :class="[currentSection === 'vitals' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/vitals" style="text-decoration: none; text-decoration-line: none;">Vitals</a>
          <a :class="[currentSection === 'queries' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/queries" style="text-decoration: none; text-decoration-line: none;">Queries</a>
          <a :class="[currentSection === 'policies' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/policies" style="text-decoration: none; text-decoration-line: none;">Policies</a>
          <a :class="[currentSection === 'software' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/app-library" style="text-decoration: none; text-decoration-line: none;">Software</a>
          <a :class="[currentSection === 'tables' ? 'active' : '']" purpose="docs-top-nav-menu-link" href="/tables" style="text-decoration: none; text-decoration-line: none;">Data tables</a>
        </div>
      </div>
      <div>
        <div purpose="nav-bar-search" id="docsearch-query" class="d-flex" v-if="algoliaPublicKey">
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
  mounted: async function(){
    let filterForSearch = {};
    if(this.searchFilter){
      filterForSearch = {
        'facetFilters': [`section:${this.searchFilter}`]
      };
    }
    if(this.algoliaPublicKey) {
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search',
        debug: false,
        searchParameters: filterForSearch,
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
