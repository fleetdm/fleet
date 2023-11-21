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
      currentTweetPage: 0,
      numberOfTweetCards: 0,
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
          <a target="_blank" herf="https://twitter.com/Linktree_"><img width="119" height="24" alt="Linktree logo" src="/images/social-proof-linktree-logo-119x24@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">I love the steady and consistent delivery of features that help teams work how they want to work, not how your product dictates they work. ❤️</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Daniel Grzelak</p>
            <p class="m-0">CISO of <a target="_blank" herf="https://twitter.com/Linktree_">Linktree</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/Uber"><img width="87" height="38" alt="Uber logo" src="/images/social-proof-logo-uber-87x38@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">Exciting. This is a team that listens to feedback.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Erik Gomez</p>
            <p class="m-0">Staff Software Engineer <a target="_blank" href="https://twitter.com/Uber">@Uber</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/lyft"><img height="38" alt="Lyft logo" src="/images/social-proof-logo-lyft-145x103@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">Keeping up with the latest issues in endpoint security is a never-ending task, because engineers have to regularly ensure every laptop and server is still sufficiently patched and securely configured. The problem is, software vendors release new versions all the time, and no matter how much you lock it down, end users find ways to change things,</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Nico Waisman</p>
            <p class="m-0">CISO of <a target="_blank" href="https://twitter.com/lyft">Lyft</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <p class="pb-2 mb-1">Fleet has been highly effective for our needs. We appreciate your team for always being so open to hearing our feedback.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Kenny Botelho</p>
            <p class="m-0"><a target="_blank" href="https://github.com/kennyb-222">@kennyb-222</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/Square"><img width="131" height="38" alt="Square logo" src="/images/social-proof-logo-square-131x38@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">Mad props to how easy making a deploy pkg of Orbit was. I wish everyone made stuff that easy.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Wesley Whetstone</p>
            <p class="m-0">CPE <a target="_blank" href="https://twitter.com/Square">@Square</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/atlassian"><img width="162" height="20" alt="Atlassian logo" src="/images/social-proof-logo-atlassian-162x20@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1"><a href="https://twitter.com/hashtag/fleet">#Fleet</a>’s come a long way - to now being the top open-source <a href="https://twitter.com/hashtag/fleet">#osquery</a> manager. Just in the past 6 months.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Brendan Shaklovitz</p>
            <p class="m-0">Senior SRE <a target="_blank" href="https://twitter.com/atlassian">@Atlassian</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/osquery"><img width="140" height="36" alt="osquery logo" src="/images/social-proof-logo-osquery-140x36@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">It’s great to see the new release of Fleet containing some really cool new features that make <a href="https://twitter.com/osquery">@osquery</a> much more usable in practical environments. I’m really impressed with the work that <a href="https://twitter.com/thezachw">@thezachw</a> and crew are doing at <a href="https://twitter.com/fleetctl">@fleetctl</a>.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Mike Arpaia</p>
            <p class="m-0">Creator of <a target="_blank" href="https://twitter.com/osquery">@osquery</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/Wayfair"><img width="136" height="32" alt="Wayfair logo" src="/images/social-proof-logo-wayfair-136x32@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1"><a href="https://twitter.com/hashtag/osquery">#osquery</a> is one of the best tools out there and <a href="https://twitter.com/hashtag/fleetdm">#fleetdm</a> makes it even better. Highly recommend it if you want to monitor, detect and investigate threats on a scale and also for infra/sys admin.</p>
        <p>I have used it on 15k servers and it’s really scalable.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Ahmed Elshaer</p>
            <p class="m-0">DFIR, Blue Teaming, SecOps <a target="_blank" href="https://twitter.com/Wayfair">@wayfair</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://twitter.com/comcast"><img width="107" height="38" alt="Comcast logo" src="/images/social-proof-logo-comcast-107x38@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">With the power of osquery, you need a scalable & resilient platform to manage your workloads. Fleet is the "just right" open-source, enterprise grade solution.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Abubakar Yousafzai</p>
            <p class="m-0">Security Software Development & Engineering <a target="_blank" href="https://twitter.com/comcast">@Comcast</a></p>
          </div>
        </div>
      </div>

      <div purpose="tweet-card" class="card">
        <div class="mb-4">
          <a target="_blank" href="https://www.linkedin.com/company/deloitte/"><img width="166" height="36" alt="Deloitte logo" src="/images/logo-deloitte-166x36@2x.png"/></a>
        </div>
        <p class="pb-2 mb-1">One of the best teams out there to go work for and help shape security platforms.</p>
        <div class="row px-3 pt-2">
          <div>
            <p class="font-weight-bold m-0">Dhruv Majumdar</p>
            <p class="m-0">Director Of Cyber Risk & Advisory <a href="https://www.linkedin.com/company/deloitte/">@Deloitte</a></p>
          </div>
        </div>
      </div>
    </div>
    <div purpose="" class="mx-auto d-flex flex-row justify-content-center">
      <nav aria-label="..." >
        <ul purpose="tweets-page-indicator" class="pagination pagination-sm" v-if="numberOfTweetPages > 1">
          <li class="page-item" :class="[currentTweetPage === index ? 'selected' : '']" v-for="(pageNumber, index) in numberOfTweetPages" @click="scrollTweetsDivToPage(index)"></li>
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
      // Get the width of the first tweet card.
      let firstTweetCardDiv = document.querySelector('div[purpose="tweet-card"]');
      this.tweetCardWidth = firstTweetCardDiv.clientWidth + 16;
      // Find out how may entire cards can fit on the screen.
      this.numberOfTweetsPerPage = Math.floor(window.innerWidth / this.tweetCardWidth);
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
      let tweetPageWidth;
      if(this.numberOfTweetPages === 2 && this.numberOfTweetsPerPage > 3){
        tweetPageWidth = (this.tweetCardWidth - 16);
      } else {
        tweetPageWidth = (this.tweetCardWidth - 16) * this.numberOfTweetsPerPage;
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
      if(page !== this.numberOfTweetPages - 1){
        tweetsDiv.scrollBy(amountToScroll, 0);
      } else {
        tweetsDiv.scrollBy(tweetsDiv.scrollWidth, 0);
      }
    },


  }
});
