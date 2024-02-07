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
      currentVisibleTweetCard: 0,
      syncing: false,
      modal: '',
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="d-flex flex-column">
    <div purpose="previous-page-indicator" @click="clickPreviousPage()" v-if="showPreviousPageButton"><img src="/images/testimonials-pagination-previous-48x48@2x.png"></div>
    <div purpose="next-page-indicator"  @click="clickNextPage()" v-if="showNextPageButton"><img src="/images/testimonials-pagination-next-48x48@2x.png"></div>
    <div purpose="tweets" class="d-flex flex-row flex-nowrap">
      <a purpose="tweet-card" class="card" v-for="testimonial in quotesToDisplay" target="_blank" :href="testimonial.quoteLinkUrl">
        <div purpose="logo" class="mb-4">
          <img :height="testimonial.imageHeight" v-if="testimonial.quoteImageFilename" :src="'/images/'+testimonial.quoteImageFilename"/>
        </div>
        <p purpose="quote">
          {{testimonial.quote}}
          <a purpose="video-link" v-if="testimonial.youtubeVideoUrl" @click.prevent.self="clickOpenVideoModal(testimonial.quoteAuthorName)">See the video.</a>
        </p>
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
      <iframe width="560" height="315" :src="'https://www.youtube.com/embed/'+video.embedId+'?rel=0'" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture;" allowfullscreen></iframe>
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
    this.tweetsDiv = $('div[purpose="tweets"]')[0];
    this.tweetCards = $('a[purpose="tweet-card"]');
    this.numberOfTweetCardsDisplayedOnThisPage = this.tweetCards.length;
    this.calculateHowManyFullTweetsCanBeDisplayed();
    $(window).resize(this.calculateHowManyFullTweetsCanBeDisplayed);
  },
  beforeDestroy: function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    calculateHowManyFullTweetsCanBeDisplayed: function() {
      let firstTweetCard = this.tweetCards[0];
      let nextTweetCard = this.tweetCards[1];
      let realCardWidth =  nextTweetCard.getBoundingClientRect().x - firstTweetCard.getBoundingClientRect().x;
      let viewportWidth = document.body.clientWidth;
      this.tweetCardWidth = realCardWidth;
      this.numberOfTweetsPerPage = Math.floor((viewportWidth - 120)/this.tweetCardWidth);
      this.pageWidth = realCardWidth * this.numberOfTweetsPerPage;
      if(this.numberOfTweetsPerPage >= this.numberOfTweetCardsDisplayedOnThisPage){
        $(this.tweetsDiv).addClass('mx-auto');
      } else {
        $(this.tweetsDiv).removeClass('mx-auto');
      }
      this.updatePageIndicators();
    },

    clickNextPage: async function() {
      if(!this.syncing){
        this.currentVisibleTweetCard += this.numberOfTweetsPerPage;
        if(this.currentVisibleTweetCard >= this.numberOfTweetCardsDisplayedOnThisPage - 1){
          this.currentVisibleTweetCard = this.numberOfTweetCardsDisplayedOnThisPage - 1;
        }
        if(this.numberOfTweetsPerPage === 1){
          let tweetCardToScrollTo = this.tweetCards[this.currentVisibleTweetCard];
          tweetCardToScrollTo.scrollIntoView({ inline: 'center', block: 'nearest' });
        } else {
          this.tweetsDiv.scrollLeft += this.pageWidth;
        }
        this.syncing = true;
        await setTimeout(()=>{
          this.updatePageIndicators();
        }, 100);
      }
    },

    clickPreviousPage: async function() {
      if(!this.syncing){
        this.currentVisibleTweetCard -= this.numberOfTweetsPerPage;
        if(this.currentVisibleTweetCard < 0 ){
          this.currentVisibleTweetCard = 0;
        }
        if(this.numberOfTweetsPerPage === 1){
          let tweetCardToScrollTo = this.tweetCards[this.currentVisibleTweetCard];
          tweetCardToScrollTo.scrollIntoView({ inline: 'center', block: 'nearest' });
        } else {
          this.tweetsDiv.scrollLeft -= this.pageWidth;
        }
        this.syncing = true;
        await setTimeout(()=>{
          this.updatePageIndicators();
        }, 100);
      }
    },

    updatePageIndicators: function() {
      this.showPreviousPageButton = this.currentVisibleTweetCard !== 0;
      console.log((this.currentVisibleTweetCard + this.numberOfTweetsPerPage) <= this.numberOfTweetCardsDisplayedOnThisPage - 1)
      this.showNextPageButton = (
        (this.currentVisibleTweetCard + this.numberOfTweetsPerPage) <= this.numberOfTweetCardsDisplayedOnThisPage - 1
        && this.numberOfTweetsPerPage !== this.numberOfTweetCardsDisplayedOnThisPage);
      this.syncing = false;
    },

    clickOpenVideoModal: function(modalName) {
      this.modal = _.kebabCase(modalName);
    },

    closeModal: function() {
      this.modal = undefined;
    },

  }
});
