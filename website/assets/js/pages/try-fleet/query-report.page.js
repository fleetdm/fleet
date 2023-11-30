parasails.registerPage('query-report', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    pageToDisplay: 0,
    numberOfPages: undefined,
    selectedTable: undefined,
    selectedHost: undefined,
    tableToDisplay: undefined,
    tableHeaders: undefined,
    hostToDisplayResultsFor: undefined,
    hostPlatformFriendlyName: '',
    hostInfo: {},
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    this.selectedHost = this.hostPlatform;
    this.hostInfo = this.hostDetails;
    if(this.selectedHost === 'macos'){
      this.hostPlatformFriendlyName = 'macOS';
    }
    if(this.selectedHost === 'windows'){
      this.hostPlatformFriendlyName = 'Windows';
    }
    if(this.selectedHost === 'linux'){
      this.hostPlatformFriendlyName = 'Linux';
    }
    this.numberOfPages = this.queryReportPages.length;
    this.tableToDisplay = this.tableName;
    this.selectedTable = this.tableToDisplay;
    this.hostToDisplayResultsFor = this.selectedHost;
    this.tableHeaders = [];
    if(this.numberOfPages !== 0){
      let columnsToShow =  _.keys(this.queryReportPages[this.pageToDisplay][0]);
      for(let column in columnsToShow){
        let columnName = columnsToShow[column];
        let columnDefinition = _.find(this.osqueryTableInfo.columns, {name: columnName});
        let columnInfo = {name: columnName, description: columnDefinition.description};
        this.tableHeaders.push(columnInfo);
      }
    }

  },
  mounted: async function() {
    if(this.numberOfPages > 0){
      this.addTableEdgeShadow();
      $(function () {
        $('[data-toggle="tooltip"]').tooltip()
      })
    }
  },

  watch: {
    selectedTable: function(val){
      if(val !== this.tableToDisplay){
        window.location = `/try-fleet/explore-data/${this.selectedHost}/${this.selectedTable}`;
      }
    },
    hostToDisplayResultsFor: function(val){
      if(val !== this.selectedHost){
        if(val === 'linux'){
          window.location = `/try-fleet/explore-data/linux/apparmor_events`
        } else if(val === 'windows'){
          window.location = `/try-fleet/explore-data/windows/appcompat_shims`
        } else {
          window.location = `/try-fleet/explore-data/macos/account_policy_data`
        }
      }
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    addTableEdgeShadow: function() {
      let tableContainer = document.querySelector('.table-responsive');
      if(tableContainer) {
        let isEdgeOfResultsTableVisible = tableContainer.scrollWidth - tableContainer.scrollLeft === tableContainer.clientWidth;
        if (!isEdgeOfResultsTableVisible) {
          tableContainer.classList.add('shadow-right');
        }

        tableContainer.addEventListener('scroll', (event)=>{
          let container = event.target;
          let isScrolledFullyToLeft = container.scrollLeft === 0;
          let isScrolledFullyToRight = (container.scrollWidth - container.scrollLeft <= container.clientWidth + 1);
          // Update the class on the table container based on how much the table is scrolled.
          if (isScrolledFullyToLeft) {
            container.classList.remove('shadow-both', 'shadow-left');
            container.classList.add('shadow-right');
          } else if (isScrolledFullyToRight) {
            container.classList.remove('shadow-both', 'shadow-right');
            container.classList.add('shadow-left');
          } else if(!isScrolledFullyToRight && !isScrolledFullyToLeft) {
            container.classList.remove('shadow-left', 'shadow-right');
            container.classList.add('shadow-both');
          }
        });
      }
    },
    clickChangePage: function(page){
      this.pageToDisplay = page - 1;
      let tableContainer = document.querySelector('.table-responsive');
      window.scrollTo({
        top: tableContainer.offsetTop - 90,
        left: 0,
        behavior: 'smooth',
      });
    },
    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },

  }
});
