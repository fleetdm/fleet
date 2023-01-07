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
      emailAddress: {required: true, isEmail: true},
    },
    cloudError: '',
    // Syncing / loading state
    syncing: false,
    howManyTweetsCanFitOnThisPage: 3,
    numberOfTweetPages: 2,
    currentTweetPage: 1,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {

    let usersScreenWidth = window.innerWidth - 120;

    this.howManyTweetsCanFitOnThisPage = Math.floor(usersScreenWidth/380);
    this.numberOfTweetPages = Math.floor(6/this.howManyTweetsCanFitOnThisPage);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
    scrollTweetDivHorizontally: function(page) {
      let tweetsDiv = document.getElementById('tweets');
      if(this.currentTweetPage === page){
        return;
      }
      console.log(`moving from ${this.currentTweetPage} to ${page}`);
      if(page === 1){
        tweetsDiv.scroll(0, 9000);
      } else if(page !== this.currentTweetPage){
        let amountToScrollBy = ((6 / this.numberOfTweetPages) * 380);
        console.log(amountToScrollBy)
        let pageDifference = page - this.currentTweetPage;
        console.log(pageDifference)
        amountToScrollBy = pageDifference * amountToScrollBy;
        console.log(amountToScrollBy)

        if(amountToScrollBy < 0){
          // amountToScrollBy = amountToScrollBy * -1;
          console.log(`this should scroll to the left by ${amountToScrollBy} pixels`)
          tweetsDiv.scrollBy(amountToScrollBy, 0);
        } else {
          console.log(`this should scroll to the right by ${amountToScrollBy} pixels`)
          tweetsDiv.scrollBy(amountToScrollBy, 0);
        }
      }
      this.currentTweetPage = page;
    },
    handleSubmittingForm: function() {
      // todo
    },
    submittedForm: function() {
      // todo
    },
  }
});
