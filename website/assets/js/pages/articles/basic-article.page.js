parasails.registerPage('basic-article', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    articleHasSubtitle: false,
    articleSubtitle: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
    // If the article has a subtitle (an H2 immediatly after an H1), we'll set articleSubtitle to be the text of that heading
    this.articleHasSubtitle = $('[purpose="article-content"]').find('h1 + h2');
    if(this.articleHasSubtitle.length > 0 && this.articleHasSubtitle[0].innerText) {
      this.articleSubtitle = this.articleHasSubtitle[0].innerText;
    }
    // Set counters for items in ordered lists to be the value of their "start" attribute.
    document.querySelectorAll('ol[start]').forEach((ol)=> {
      let startValue = parseInt(ol.getAttribute('start'), 10) - 1;
      ol.style.counterReset = 'custom-counter ' + startValue;
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickCopyRssLink: function(articleCategory) {
      let rssButton = $('a[purpose="rss-button"]');
      if(typeof navigator.clipboard !== 'undefined' && rssButton) {
        // If this heading has already been clicked and still has the copied class we'll just ignore this click
        if(!$(rssButton).hasClass('copied')) {
          navigator.clipboard.writeText('https://fleetdm.com/rss/'+articleCategory);
          // Add the copied class to the header to notify the user that the link has been copied.
          $(rssButton).addClass('copied');
          // Remove the copied class 5 seconds later, so we can notify the user again if they re-cick on this heading
          setTimeout(()=>{$(rssButton).removeClass('copied');}, 5000);
        }
      } else {
        window.open('https://fleetdm.com/rss/'+articleCategory, '_blank');
      }
    },
  }
});
