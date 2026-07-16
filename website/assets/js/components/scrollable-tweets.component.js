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
      tweetsDiv: undefined,
      tweetCards: undefined,
      pageWidth: undefined,
      numberOfTweetCardsDisplayedOnThisPage: undefined,
      showPreviousPageButton: false,
      showNextPageButton: true,
      numberOfTweetsPerPage: 0,
      syncing: false,
      firstCardPosition: 0,
      modal: '',
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="d-flex flex-column">

    <div purpose="tweets" class="d-flex flex-row flex-nowrap">
    <div purpose="previous-page-indicator" @click="clickPreviousPage()" v-if="showPreviousPageButton"><img src="/images/testimonials-pagination-previous-48x48@2x.png"></div>
    <div purpose="next-page-indicator"  @click="clickNextPage()" v-if="showNextPageButton"><img src="/images/testimonials-pagination-next-48x48@2x.png"></div>
      <a purpose="tweet-card" class="card" v-for="testimonial in quotesToDisplay" target="_blank" :href="testimonial.quoteLinkUrl" no-icon>
        <div purpose="logo" class="mb-4">
          <img :height="testimonial.imageHeight" v-if="testimonial.quoteImageFilename" :src="'/images/'+testimonial.quoteImageFilename"/>
        </div>
        <div purpose="quote-container" :class="{ overflowing: testimonial.isQuoteOverflowing && !testimonial.isQuoteExpanded, expanded: testimonial.isQuoteExpanded }">
          <p purpose="quote">
            {{testimonial.quote}}
            <a purpose="video-link" v-if="testimonial.youtubeVideoUrl" @click.prevent.stop="clickOpenVideoModal(testimonial.quoteAuthorName)">See the video.</a>
          </p>
        </div>
        <a purpose="show-full-quote-link" v-if="testimonial.isQuoteOverflowing && !testimonial.isQuoteExpanded" @click.prevent.stop="testimonial.isQuoteExpanded = true">Show full quote</a>
        <div purpose="quote-author-info" class="d-flex flex-row align-items-center">
          <div purpose="profile-picture">
            <img :src="'/images/'+testimonial.quoteAuthorProfileImageFilename">
          </div>
          <div class="d-flex flex-column align-self-top">
            <p purpose="name" class="font-weight-bold m-0">{{testimonial.quoteAuthorName}}</p>
            <p purpose="job-title" class="m-0">{{testimonial.quoteAuthorJobTitle}}</p>
          </div>
        </div>
      </a>
    </div>
    <div v-for="video in quotesWithVideoLinks">
    <modal purpose="video-modal" v-if="modal === video.modalId" @close="closeModal()" >
      <iframe width="560" height="315" :src="'https://www.youtube-nocookie.com/embed/'+video.embedId+'?rel=0'" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture;" allowfullscreen></iframe>
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
    this.quotesToDisplay = this.testimonials.map((testimonial) =>{
      if(testimonial.youtubeVideoUrl) {
        this.quotesWithVideoLinks.push({
          modalId: _.kebabCase(testimonial.quoteAuthorName),
          embedId: testimonial.videoIdForEmbed,
        });
      }
      return Object.assign({}, testimonial, { isQuoteOverflowing: false, isQuoteExpanded: false});
    });
  },
  mounted: async function(){
    this.tweetsDiv = $('div[purpose="tweets"]')[0];
    this.tweetCards = $('a[purpose="tweet-card"]');
    try {
      this.firstCardPosition = this.tweetCards[0].getBoundingClientRect().x;
    } catch (err) {
      console.warn('Could not determine position of testimonials in scrollable-tweets component.  Using fake position, which may cause rendering issues.  Error details:',err);
      this.firstCardPosition = 0;
    }
    this.numberOfTweetCardsDisplayedOnThisPage = this.tweetCards.length;
    this.calculateHowManyFullTweetsCanBeDisplayed();
    this.checkQuoteOverflow();
    $(window).on('resize', this.calculateHowManyFullTweetsCanBeDisplayed);
    $(window).on('resize', this.checkQuoteOverflow);
    $(window).on('wheel', this.updatePageIndicators);
  },
  beforeDestroy: function() {
    $(window).off('.scrollableTweets');
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    calculateHowManyFullTweetsCanBeDisplayed: function() {
      let firstTweetCard = this.tweetCards[0];
      let nextTweetCard = this.tweetCards[1];
      try {
        this.tweetCardWidth =  nextTweetCard.getBoundingClientRect().x - firstTweetCard.getBoundingClientRect().x;
        this.numberOfTweetsPerPage = Math.floor((document.body.clientWidth - this.firstCardPosition)/this.tweetCardWidth);
      } catch (err) {
        console.warn('Could not determine "per page" and "card width" for testimonials in scrollable-tweets component.  Using fake position, which may cause rendering issues.  Error details:',err);
        this.numberOfTweetsPerPage = 1;
        this.tweetCardWidth = 100;
      }
      if(this.numberOfTweetsPerPage < 1){
        this.numberOfTweetsPerPage = 1;
      }
      this.pageWidth = this.tweetCardWidth * this.numberOfTweetsPerPage;
      if(this.numberOfTweetsPerPage >= this.numberOfTweetCardsDisplayedOnThisPage){
        $(this.tweetsDiv).addClass('mx-auto');
      } else {
        $(this.tweetsDiv).removeClass('mx-auto');
      }
      this.updatePageIndicators();
    },

    clickNextPage: async function() {
      if(!this.syncing){
        this.tweetsDiv.scrollLeft += this.pageWidth;
        await setTimeout(()=>{
          this.updatePageIndicators();
        }, 600);
      }
    },

    clickPreviousPage: async function() {
      if(!this.syncing){
        this.tweetsDiv.scrollLeft -= this.pageWidth;
        await setTimeout(()=>{
          this.updatePageIndicators();
        }, 600);
      }
    },

    updatePageIndicators: function() {
      this.syncing = false;
      this.showPreviousPageButton = this.tweetsDiv.scrollLeft > (this.firstCardPosition * 0.5);
      this.showNextPageButton = (this.tweetsDiv.scrollWidth - this.tweetsDiv.scrollLeft - this.tweetsDiv.clientWidth) >= this.tweetCardWidth * .25;
    },

    clickOpenVideoModal: function(modalName) {
      this.modal = _.kebabCase(modalName);
    },

    closeModal: function() {
      this.modal = undefined;
    },

    checkQuoteOverflow: function() {
      // Check if a card's quote exceeds the set max-height, and set isQuoteOverflowing values on quote cards.
      let containers = this.$el.querySelectorAll('[purpose="quote-container"]');
      let minHiddenHeightToTruncate = 40;// « The number of pixels a quote must exceed the height of a parent container by for a quote to be truncated.
      containers.forEach((el, i) => {
        let quote = this.quotesToDisplay[i];
        if (quote && !quote.isQuoteExpanded) {
          let hiddenHeight = el.scrollHeight - el.clientHeight;
          if (hiddenHeight > minHiddenHeightToTruncate) {
            quote.isQuoteOverflowing = true;
          } else if (hiddenHeight > 1) {
            // Overflows the cap, but only barely — show the whole quote (no button).
            quote.isQuoteOverflowing = false;
            quote.isQuoteExpanded = true;
          }
        }
      });
    },

  }
});
