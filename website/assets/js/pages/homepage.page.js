parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
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
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

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

    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },
  }
});
