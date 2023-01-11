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
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function () {
    return {
      currentTweetPage: 1,
      numberOfTweetCards: 6,
      numberOfTweetPages: 0,
      numberOfTweetsPerPage: 0,
      tweetCardWidth: 0,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="d-flex flex-column">
    <div purpose="tweets" class="d-flex flex-row flex-nowrap">
      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/Uber"><img width="87" height="38" alt="Uber logo" src="/images/social-proof-logo-uber-87x38@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">Exciting. This is a team that listens to feedback.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Erik Gomez</p>
            <p class="m-0">Staff Software Engineer <a href="https://twitter.com/Uber">@Uber</a></p>
          </div>
        </div>
      </div>
      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/Square"><img width="131" height="38" alt="Square logo" src="/images/social-proof-logo-square-131x38@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">Mad props to how easy making a deploy pkg of Orbit was. I wish everyone made stuff that easy.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Wesley Whetstone</p>
            <p class="m-0">CPE <a href="https://twitter.com/Square">@Square</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/atlassian"><img width="162" height="20" alt="Atlassian logo" src="/images/social-proof-logo-atlassian-162x20@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1"><a href="https://twitter.com/hashtag/fleet">#Fleet</a>’s come a long way - to now being the top open-source <a href="https://twitter.com/hashtag/fleet">#osquery</a> manager. Just in the past 6 months.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Brendan Shaklovitz</p>
            <p class="m-0">Senior SRE <a href="https://twitter.com/atlassian">@Atlassian</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/osquery"><img width="140" height="36" alt="osquery logo" src="/images/social-proof-logo-osquery-140x36@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">It’s great to see the new release of Fleet containing some really cool new features that make <a href="https://twitter.com/osquery">@osquery</a> much more usable in practical environments. I’m really impressed with the work that <a href="https://twitter.com/thezachw">@thezachw</a> and crew are doing at <a href="https://twitter.com/fleetctl">@fleetctl</a>.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Mike Arpaia</p>
            <p class="m-0">Creator of <a href="https://twitter.com/osquery">@osquery</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/Wayfair"><img width="136" height="32" alt="Wayfair logo" src="/images/social-proof-logo-wayfair-136x32@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1"><a href="https://twitter.com/hashtag/osquery">#osquery</a> is one of the best tools out there and <a href="https://twitter.com/hashtag/fleetdm">#fleetdm</a> makes it even better. Highly recommend it if you want to monitor, detect and investigate threats on a scale and also for infra/sys admin.</p>
        <p>I have used it on 15k servers and it’s really scalable.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Ahmed Elshaer</p>
            <p class="m-0">DFIR, Blue Teaming, SecOps <a href="https://twitter.com/Wayfair">@wayfair</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a href="https://twitter.com/comcast"><img width="107" height="38" alt="Comcast logo" src="/images/social-proof-logo-comcast-107x38.png"/></a>
        </div>
        <p class="pb-2 mb-1">With the power of osquery, you need a scalable & resilient platform to manage your workloads. Fleet is the "just right" open-source, enterprise grade solution.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Abubakar Yousafzai</p>
            <p class="m-0">Security Software Development & Engineering <a href="https://twitter.com/comcast">@Comcast</a></p>
          </div>
        </div>
      </div>
      <div purpose="tweet-cards-right-padding">
      </div>
    </div>
    <div purpose="" class="mx-auto">
      <nav aria-label="..." >
        <ul purpose="tweets-page-indicator" class="pagination pagination-sm">
          <li class="page-item" :class="[currentTweetPage === pageNumber ? 'selected' : '']" v-for="pageNumber in numberOfTweetPages" @click="scrollTweetsDivToPage(pageNumber)"></li>
        </ul>
      </nav>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    await this.updateNumberOfTweetPages(); // Update the number of pages for the tweet page indicator.
    window.addEventListener('wheel', this.updateCurrentPageIndicator); // Add a mouse wheel event listener to update the tweet page indicator when a user scrolls the div.
    window.addEventListener('resize', this.updateNumberOfTweetPages); // Add an event listener to update the number of tweet pages based on how many tweet cards can fit on the screen.
  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    updateNumberOfTweetPages: async function() {
      // Get the first tweet card in the tweets div.
      let firstTweetCardDiv = $('div[purpose="tweet-card"]')[0];
      // Update the tweetCardWidth to have 16 pixels of padding.
      this.tweetCardWidth = firstTweetCardDiv.clientWidth + 16;
      let usersScreenWidth = window.innerWidth;
      // Get the number of tweets that can be visible on the user's screen
      this.numberOfTweetsPerPage = Math.floor(usersScreenWidth/this.tweetCardWidth);
      // Divide the number of tweet cards by the number of tweets that can fit on a users screen
      this.numberOfTweetPages = Math.ceil(this.numberOfTweetCards / this.numberOfTweetsPerPage);
      await this.forceRender();
    },
    updateCurrentPageIndicator: function() {
      // Get the tweets div.
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      // Get the amount the tweets div has been scrolled to the left.
      let currentTweetDivScrollAmount = tweetsDiv.scrollLeft;
      // Divide the current amount scrolled by the width of a tweet card, and divide that value by how many tweet cards we can show on a page.
      let pageCurrentlyViewed = ((currentTweetDivScrollAmount) / this.tweetCardWidth) / this.numberOfTweetsPerPage;
      let pageToIndicate = Math.ceil(pageCurrentlyViewed + 1);
      // Update the currentTweetPage value
      this.currentTweetPage = pageToIndicate;
    },
    scrollTweetsDivToPage: function(page) {
      let tweetsDiv = document.querySelector('div[purpose="tweets"]');
      if(page === this.currentTweetPage){ // If the page it is currently on is selected, do nothing.
        return;
      } else if(page === 1){// If the first page was selected, scroll the tweets div to the starting position.
        tweetsDiv.scroll(0, 9000);
      } else if(page !== this.currentTweetPage) { // If any other page was selected, scroll the tweets div.
        // Get the amount we need to scroll for a single page.
        let amountToScrollBy = ((6 / this.numberOfTweetPages) * this.tweetCardWidth);
        // Get the number of pages we're moving
        let pageDifference = page - this.currentTweetPage;
        // Multiply the amount to scroll by the number of pages we're scrolling
        amountToScrollBy = pageDifference * amountToScrollBy;
        // Scroll the Tweets div
        tweetsDiv.scrollBy(amountToScrollBy, 0);
      }
      // Set the current page.
      this.currentTweetPage = page;
    },

  }
});
