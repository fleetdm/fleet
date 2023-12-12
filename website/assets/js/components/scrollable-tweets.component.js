/**
 * <scrollable-tweets>
 * -----------------------------------------------------------------------------
 * A horizontally scrolling row of tweets with an auto-updating page indicator
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('scrollableTweets', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'testimonials'
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function () {
    return {
      quotesToDisplay: [],
      quotesWithVideoLinks: [],
      currentTweetPage: 0,
      numberOfTweetCards: 0,
      numberOfTweetPages: 0,
      numberOfTweetsPerPage: 0,
      tweetCardWidth: 0,
      tweetPageWidth: 0,
      screenSize: 0,
      scrolledAmount: 0,
      scrollableAmount: 0,
      modal: ''
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="d-flex flex-column">
    <div purpose="tweets" class="d-flex flex-row flex-nowrap">
      <div purpose="tweet-card" class="card" v-for="testimonial in quotesToDisplay">
        <div class="mb-4" v-if="testimonial.quoteImagePathInAssetsFolder">
          <a target="_blank" :href="testimonial.quoteImageLinkUrl"><img :height="testimonial.imageHeight" :src="'/images/'+testimonial.quoteImagePathInAssetsFolder"/></a>
        </div>
        <p class="pb-2 mb-1">
          {{testimonial.quote}}
          <span purpose="video-link" v-if="testimonial.youtubeVideoUrl" @click="clickOpenVideoModal(testimonial.quoteAuthorName)">See the video.</span>
        </p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">{{testimonial.quoteAuthorName}}</p>
            <p class="m-0">
              {{testimonial.quoteAuthorJobTitle}}
              <a target="_blank" :href="testimonial.quoteAuthorSocialLink">{{testimonial.quoteAuthorSocialHandle}}
              </a>
              </p>
          </div>
        </div>
      </div>
    </div>
    <div purpose="page-indictator-container" class="mx-auto d-flex flex-row justify-content-center">
      <div class="d-flex flex-row flex-wrap align-items-center justify-content-center" v-if="numberOfTweetPages > 1 && numberOfTweetPages <= numberOfTweetCards">
        <div purpose="page-indicator" :class="[currentTweetPage === index ? 'selected' : '']" v-for="(pageNumber, index) in numberOfTweetPages" @click="scrollTweetsDivToPage(index)"></div>
      </div>
    </div>
    <div v-for="video in quotesWithVideoLinks">
    <modal purpose="video-modal" v-if="modal === video.modalId" @close="closeModal()" >
      <iframe width="560" height="315" :src="'https://www.youtube.com/embed/'+video.embedId+'?controls=0'" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
    </modal>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(!this.testimonials){
      throw new Error('Incomplete usage of <scrollable-tweets>:  Please pass in a `testimonials` prop (an array of testimonials from sails.config.builtStaticContent.testimonials).  For example: `<scrollable-tweets :testimonials="testimonials">`');
    }
    if(!_.isArray(this.testimonials)){
      throw new Error('Incomplete usage of <scrollable-tweets>:  The `testimonials` prop provided is an invalid type. Please provide an array of testimonial values.');
    }
    this.quotesToDisplay = _.clone(this.testimonials);
    for(let quote of this.testimonials){
      if(quote.youtubeVideoUrl){
        this.quotesWithVideoLinks.push({
          modalId: _.kebabCase(quote.quoteAuthorName),
          embedId: quote.videoIdForEmbed,
        });
      }
    }

  },
  mounted: async function(){
    let tweetsDiv = document.querySelector('div[purpose="tweets"]');
    let tweetCards = document.querySelectorAll('div[purpose="tweet-card"]');
    this.numberOfTweetCards = tweetCards.length;
    await this.updateNumberOfTweetPages(); // Update the number of pages for the tweet page indicator.
    tweetsDiv.addEventListener('scroll', this.updatePageIndicator, {passive: true}); // Add a scroll event listener to update the tweet page indicator when a user scrolls the div.
    window.addEventListener('resize', this.updateNumberOfTweetPages); // Add an event listener to update the number of tweet pages based on how many tweet cards can fit on the screen.
  },
  beforeDestroy: function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    updateNumberOfTweetPages: async function() {
      this.screenSize = window.innerWidth;
      // Get the width of the first tweet card.
      let firstTweetCardDiv = document.querySelector('div[purpose="tweet-card"]');
      this.tweetCardWidth = firstTweetCardDiv.clientWidth + 16;
      // Find out how may entire cards can fit on the screen.
      this.numberOfTweetsPerPage = Math.floor(this.screenSize / this.tweetCardWidth);
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      this.scrollableAmount = tweetsDiv.scrollWidth - this.screenSize;
      // Find out how many pages of tweet cards there will be.
      this.numberOfTweetPages = Math.ceil(this.numberOfTweetCards / this.numberOfTweetsPerPage);
      // Update the current page indicator.
      this.updatePageIndicator();
      await this.forceRender();
    },

    updatePageIndicator: function() {
      // Get the tweets div.
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      // Find out the width of a page of tweet cards
      this.scrolledAmount = tweetsDiv.scrollLeft;
      this.tweetPageWidth = this.tweetCardWidth * this.numberOfTweetsPerPage;
      // Set the maximum number of pages as the maximum value
      let currentPage = Math.min(Math.round(tweetsDiv.scrollLeft / this.tweetPageWidth), (this.numberOfTweetPages - 1));
      if(tweetsDiv.scrollLeft === this.scrollableAmount) {
        currentPage = this.numberOfTweetPages - 1;
      }
      // Update the page indicator
      this.currentTweetPage = currentPage;
    },
    clickOpenVideoModal: function(modalName) {
      this.modal = _.kebabCase(modalName);
    },

    closeModal: function() {
      this.modal = undefined;
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
      if(page !== this.numberOfTweetPages - 1){
        tweetsDiv.scrollBy(amountToScroll, 0);
      } else {
        tweetsDiv.scrollBy(tweetsDiv.scrollWidth, 0);
      }
    },


  }
});
