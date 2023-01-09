parasails.registerPage('fleet-mdm', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      fullName: {required: true },
      emailAddress: {required: true, isEmail: true},
      jobTitle: {required: true },
    },
    cloudError: '',
    // Syncing / loading state
    syncing: false,
    showSignupFormSuccess: false,
    // Modal

    modal: '',

    // Page indicatior for the scrollable tweet cards
    currentTweetPage: 1,
    numberOfTweetPages: 2,
    howManyTweetsCanFitOnThisPage: 3,
    tweetDivPaddingWidth: 0,
  },
  computed: {
    tweetCardWidth: function() {
      let firstTweetCardDiv = $('div[purpose="tweet-card"]')[0];
      return firstTweetCardDiv.clientWidth + 16;
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    await this.updateNumberOfTweetPages(); // Update the number of pages for the tweet page indicator.
    window.addEventListener('mousewheel', this.updateCurrentTweetsPage); // Add a mouse wheel event listener to update the tweet page indicator when a user scrolls the div.
    window.addEventListener('resize', this.updateNumberOfTweetPages); // Add an event listener to updat ethe number of tweet pages based on how many tweet cards can fit on the screen.
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
    updateCurrentTweetsPage: async function() {
      let tweetsDiv = $('div[purpose="tweets"]')[0];
      let currentTweetDivScrollAmount = tweetsDiv.scrollLeft;
      let pageThisShouldBe = ((currentTweetDivScrollAmount) / this.tweetCardWidth);
      let pageToIndicate = Math.round(pageThisShouldBe) > 0 ? Math.round(pageThisShouldBe) + 1 : 1;
      if(pageToIndicate > this.numberOfTweetPages){
        pageToIndicate = this.numberOfTweetPages;
      }
      this.currentTweetPage = pageToIndicate;
    },
    updateNumberOfTweetPages: async function() {
      let tweetsPaddingAmount = $('div[purpose="tweets"]').css('padding-left');
      this.tweetDivPaddingWidth = eval(tweetsPaddingAmount.split('px')[0]);
      let usersScreenWidth = window.innerWidth;
      this.howManyTweetsCanFitOnThisPage = Math.floor(usersScreenWidth/this.tweetCardWidth);
      this.numberOfTweetPages = Math.floor(6/this.howManyTweetsCanFitOnThisPage);
      if(this.numberOfTweetPages === Infinity) {
        this.numberOfTweetPages = 6;
      }
      await this.forceRender();
    },
    scrollTweetDivHorizontally: function(page) {
      let tweetsDiv = document.getElementById('tweets');
      if(this.currentTweetPage === page){
        return;
      }
      if(page === 1){
        tweetsDiv.scroll(0, 9000);
      } else if(page !== this.currentTweetPage) {
        let amountToScrollBy = ((6 / this.numberOfTweetPages) * this.tweetCardWidth);
        let pageDifference = page - this.currentTweetPage;
        amountToScrollBy = pageDifference * amountToScrollBy;
        tweetsDiv.scrollBy(amountToScrollBy, 0);
      }
      this.currentTweetPage = page;
    },
    clickOpenSignupModal: function() {
      this.modal = 'beta-signup';
    },
    closeModal: function () {
      this.modal = '';
    },
    submittedForm: function() {
      this.showSignupFormSuccess = true;
    },
  }
});
