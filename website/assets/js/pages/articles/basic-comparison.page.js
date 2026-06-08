parasails.registerPage('basic-comparison', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    subtopics: [],
    lastScrollTop: 0,
    scrollDistance: 0,
    tableHeadersForMobileComparisonTable: []
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    this.subtopics = (() => {
      let subtopics = $('[purpose="article-content"]').find('h2.markdown-heading').map((_, el) => el);
      subtopics = $.makeArray(subtopics).map((subheading) => {
        return {
          title: subheading.innerText,
          url: $(subheading).find('a.markdown-link').attr('href'),
        };
      });
      return subtopics;
    })();
    $('.table').find('thead th').each((index, el)=>{
      console.log(el);
      this.tableHeadersForMobileComparisonTable.push($(el).text().trim());
    });
    $('.table').find('tbody tr').each((index, el)=>{
      $(el).find('td').each((index, el)=>{
        let headerForThisColumn = this.tableHeadersForMobileComparisonTable[index];
        if (headerForThisColumn) {
          $(el).attr('data-label', headerForThisColumn);
        }
      });
    });
    // Add an event listener to add a class to the right sidebar when the header is hidden.
    window.addEventListener('scroll', this.handleScrollingInArticle);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleScrollingInArticle: function () {
      let rightNavBar = document.querySelector('div[purpose="right-sidebar"]');
      let scrollTop = window.pageYOffset;
      let windowHeight = window.innerHeight;
      // Add/remove the 'header-hidden' class to the right sidebar to scroll it upwards with the website's header.
      if (rightNavBar) {
        if (scrollTop > this.scrollDistance && scrollTop > windowHeight * 1.5) {
          rightNavBar.classList.add('header-hidden');
          this.lastScrollTop = scrollTop;
        } else if(scrollTop < this.lastScrollTop - 60) {
          rightNavBar.classList.remove('header-hidden');
          this.lastScrollTop = scrollTop;
        }
      }
      this.scrollDistance = scrollTop;
    },
  }
});
