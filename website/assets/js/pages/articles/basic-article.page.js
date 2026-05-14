parasails.registerPage('basic-article', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    articleHasSubtitle: false,
    articleSubtitle: undefined,
    subtopics: [],
    lastScrollTop: 0,
    scrollDistance: 0,
    isIpadOS: false,
    showLastUpdated: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (this.thisPage && this.thisPage.lastModifiedAt && this.thisPage.meta && this.thisPage.meta.publishedOn) {
      var publishedTimestamp = new Date(this.thisPage.meta.publishedOn).getTime();
      var oneDayMs = 86400000;
      if (this.thisPage.lastModifiedAt - publishedTimestamp > oneDayMs) {
        this.showLastUpdated = true;
      }
    }
  },
  mounted: async function() {
    // Set a flag to determine whether or not this is an ipad. (Used to show/hide an embeded PDF)
    if(navigator.maxTouchPoints > 1 && bowser.mac) {
      this.isIpadOS = true;
    }
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

    let headingsOnThisPage = $('[purpose="article-content"]').find(':header');
    for(let key in Object.values(headingsOnThisPage)){
      let heading = headingsOnThisPage[key];
      // Find the child <a> element
      let linkElementNestedInThisHeading = _.first($(heading).find('a.markdown-link'));
      $(linkElementNestedInThisHeading).click(()=> {
        if(typeof navigator.clipboard !== 'undefined') {
          // If this heading has already been clicked and still has the copied class we'll just ignore this click
          if(!$(heading).hasClass('copied')){
            // If the link's href is missing, we'll copy the current url (and remove any hashes) to the clipboard instead
            if(linkElementNestedInThisHeading.href) {
              navigator.clipboard.writeText(linkElementNestedInThisHeading.href);
            } else {
              navigator.clipboard.writeText(heading.baseURI.split('#')[0]);
            }
            // Add the copied class to the header to notify the user that the link has been copied.
            $(heading).addClass('copied');
            // Remove the copied class 5 seconds later, so we can notify the user again if they re-cick on this heading
            setTimeout(()=>{$(heading).removeClass('copied');}, 5000);
          }
        }
      });
    }
    // https://github.com/sailshq/sailsjs.com/blob/7a74d4901dcc1e63080b502492b03fc971d3d3b2/assets/js/functions/sails-website-actions.js#L177-L239
    (function highlightThatSyntax(){
      $('pre code').each((i, block) => {
        window.hljs.highlightElement(block);
      });

      // Make sure the <pre> tags whose code isn't being highlighted
      // has that nice muted look we like.
      $('.nohighlight').each(function() {
        var $codeBlock = $(this);
        $codeBlock.closest('pre').addClass('muted');
      });
      // Also make sure the 'usage' (and 'usage-*') code blocks have special styles.
      $('.usage,.usage-exec').each(function() {
        var $codeBlock = $(this);
        $codeBlock.closest('pre').addClass('usage-wrapper');
      });

      // Now let's make the `function` keywords blue like in sublime.
      $('.hljs-keyword').each(function() {
        var $highlightedKeyword = $(this);
        if($highlightedKeyword.text() === 'function') {
          $highlightedKeyword.removeClass('hljs-keyword');
          $highlightedKeyword.addClass('hljs-function-keyword');
        }
      });
    })();

    // Add an event listener to add a class to the right sidebar when the header is hidden.
    window.addEventListener('scroll', this.handleScrollingInArticle);

    if(this.algoliaPublicKey) {// Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the handbook search will be disabled.
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search',
        debug: false,
        clickAnalytics: true,
        searchParameters: {
          facetFilters: ['section:articles']
        },
      });
      // For mobile search
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#mobile-docsearch',
        placeholder: 'Search',
        debug: false,
        clickAnalytics: true,
        searchParameters: {
          facetFilters: ['section:articles']
        },
      });
    }
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
    clickGotoStart: function() {
      this.goto('/register');
    },
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
