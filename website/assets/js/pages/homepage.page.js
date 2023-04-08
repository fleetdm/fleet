parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Main syncing/loading state for this page.
    syncing: false,

    // Form data
    formData: {
      subscribeTo: 'releases'
    },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      emailAddress: {isEmail: true, required: true},
    },

    // Server error state for the form
    cloudError: '',

    // Success state when form has been submitted
    cloudSuccess: false,
    showAllTweets: false,
    modal: undefined,

    currentTweetPage: 0,
    numberOfTweetCards: 6,
    numberOfTweetPages: 0,
    numberOfTweetsPerPage: 0,
    tweetCardWidth: 0,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    await this.updateNumberOfTweetPages(); // Update the number of pages for the tweet page indicator.
    const tweetsDiv = document.querySelector('div[purpose="tweets"]');
    tweetsDiv.addEventListener('scroll', this.updatePageIndicator, {passive: true}); // Add a scroll event listener to update the tweet page indicator when a user scrolls the div.
    window.addEventListener('resize', this.updateNumberOfTweetPages); // Add an event listener to update the number of tweet pages based on how many tweet cards can fit on the screen.
    // window.addEventListener('scroll', this.scrollBackground); // Add an event listener to update the number of tweet pages based on how many tweet cards can fit on the screen.


  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    // scrollBackground: function() {
    //   let farBackgroundContainer = document.querySelector('div[purpose="cloud-city-bottom-banner"]');
    //   let innerBackgroundContainer = document.querySelector('div[purpose="background"]');
    //   let pageHeight = $('[purpose=page-wrap]')[0].clientHeight;
    //   let howFarToTheBottom = window.pageYOffset - pageHeight + window.innerHeight;

    //   if (howFarToTheBottom > -253) {
    //     farBackgroundContainer.style.backgroundAttachment = 'scroll';
    //     farBackgroundContainer.style.backgroundPosition = 'center bottom';
    //     innerBackgroundContainer.style.backgroundAttachment = 'scroll';
    //     innerBackgroundContainer.style.backgroundPosition = 'center bottom';
    //   } else if (howFarToTheBottom < -525) {
    //     farBackgroundContainer.style.backgroundAttachment = 'scroll';
    //     farBackgroundContainer.style.backgroundPosition = 'center top';
    //     innerBackgroundContainer.style.backgroundAttachment = 'scroll';
    //     innerBackgroundContainer.style.backgroundPosition = 'center top';
    //   } else {
    //     if (howFarToTheBottom < -363) {
    //       innerBackgroundContainer.style.backgroundAttachment = 'scroll';
    //       innerBackgroundContainer.style.backgroundPosition = 'center top';
    //     } else {
    //       innerBackgroundContainer.style.backgroundAttachment = 'fixed';
    //       innerBackgroundContainer.style.backgroundPosition = 'center bottom';
    //     }

    //     farBackgroundContainer.style.backgroundAttachment = 'fixed';
    //     farBackgroundContainer.style.backgroundPosition = 'center bottom';
    //   }
    // },


    updateNumberOfTweetPages: async function() {
      // Get the width of the first tweet card.
      let firstTweetCardDiv = document.querySelector('div[purpose="tweet-card"]');
      this.tweetCardWidth = firstTweetCardDiv.clientWidth + 16;
      // Find out how may entire cards can fit on the screen.
      this.numberOfTweetsPerPage = Math.floor(window.innerWidth / this.tweetCardWidth);
      // Find out how many pages of tweet cards there will be.
      this.numberOfTweetPages = Math.ceil(this.numberOfTweetCards / this.numberOfTweetsPerPage);
      if(this.numberOfTweetPages < 1){
        this.numberOfTweetPages = 1;
      } else if (this.numberOfTweetPages > this.numberOfTweetCards) {
        this.numberOfTweetPages = this.numberOfTweetCards;
      }
      // Update the current page indicator.
      this.updatePageIndicator();
      await this.forceRender();
    },

    updatePageIndicator: function() {
      // Get the tweets div.
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      // Find out the width of a page of tweet cards
      let tweetPageWidth;
      if(this.numberOfTweetPages === 2 && this.numberOfTweetsPerPage > 3){
        tweetPageWidth = this.tweetCardWidth;
      } else {
        tweetPageWidth = this.tweetCardWidth * this.numberOfTweetsPerPage;
      }
      // Set the maximum number of pages as the maximum value
      let currentPage = Math.min(Math.round(tweetsDiv.scrollLeft / tweetPageWidth), (this.numberOfTweetPages - 1));
      // Update the page indicator
      this.currentTweetPage = currentPage;
    },

    scrollTweetsDivToPage: function(page) {
      // Get the tweets div.
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      // Find out the width of a page of tweet cards
      let pageWidth = this.tweetCardWidth * this.numberOfTweetsPerPage;
      // Figure out how much distance we're expecting to scroll.
      let baseAmountToScroll = (page - this.currentTweetPage) * pageWidth;
      // Find out the actual distance the div has been scrolled
      let amountCurrentPageHasBeenScrolled = tweetsDiv.scrollLeft - (this.currentTweetPage * pageWidth);
      // subtract the amount the current page has been scrolled from the baseAmountToScroll
      let amountToScroll = baseAmountToScroll - amountCurrentPageHasBeenScrolled;
      // Scroll the div to the specified 'page'
      tweetsDiv.scrollBy(amountToScroll, 0);
    },
    clickChatButton: function() {
      // Temporary hack to open the chat
      // (there's currently no official API for doing this outside of React)
      //
      // > Alex: hey mike! if you're just trying to open the chat on load, we actually have a `defaultIsOpen` field
      // > you can set to `true` :) i haven't added the `Papercups.open` function to the global `Papercups` object yet,
      // > but this is basically what the functions look like if you want to try and just invoke them yourself:
      // > https://github.com/papercups-io/chat-widget/blob/master/src/index.tsx#L4-L6
      // > ~Dec 31, 2020
      window.dispatchEvent(new Event('papercups:open'));
    },
  }
});
