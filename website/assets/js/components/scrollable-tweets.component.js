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
    <div purpose="page-indictator-container" class="mx-auto d-flex flex-row justify-content-center">
    </div>
    <div v-for="video in quotesWithVideoLinks">
    <modal purpose="video-modal" v-if="modal === video.modalId" @close="closeModal()" >
      <iframe width="560" height="315" :src="'https://www.youtube.com/embed/'+video.embedId+'?rel=0'" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
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

  },
  beforeDestroy: function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickOpenVideoModal: function(modalName) {
      this.modal = _.kebabCase(modalName);
    },

    closeModal: function() {
      this.modal = undefined;
    },

  }
});
