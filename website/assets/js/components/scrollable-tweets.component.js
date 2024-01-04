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
        <div class="mb-4" v-if="testimonial.quoteImageFilename">
          <a target="_blank" :href="testimonial.quoteLinkUrl"><img :height="testimonial.imageHeight" :src="'/images/'+testimonial.quoteImageFilename"/></a>
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
              <a target="_blank" :href="testimonial.quoteLinkUrl">{{testimonial.quoteAuthorSocialHandle}}
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
    <div purpose="page-indictator-container" class="mx-auto d-flex flex-row justify-content-center">
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
