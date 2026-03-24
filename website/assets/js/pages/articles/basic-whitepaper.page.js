parasails.registerPage('basic-whitepaper', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: {
      emailAddress: undefined,
      firstName: undefined,
      lastName: undefined,
    },
    formRules: {
      emailAddress: {isEmail: true, required: true},
      firstName: {required: true},
      lastName: {required: true},
    },
    formDataToPrefillForLoggedInUsers: {},
    formErrors: {},
    syncing: false,
    cloudError: '',
    cloudSuccess: '',
    scrollDistance: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.me){// prefill from database
      this.formDataToPrefillForLoggedInUsers.emailAddress = this.me.emailAddress;
      this.formDataToPrefillForLoggedInUsers.firstName = this.me.firstName;
      this.formDataToPrefillForLoggedInUsers.lastName = this.me.lastName;
      this.formData = _.clone(this.formDataToPrefillForLoggedInUsers);
    }
  },
  mounted: async function() {
    this.formData.whitepaperName = this.thisPage.meta.articleTitle;

    // Add an event listener to add a class to the right sidebar when the header is hidden.
    window.addEventListener('scroll', this.handleScrollingInArticle);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    submittedDownloadForm: async function () {
      if(typeof qualified !== 'undefined') {
        qualified('saveFormData',
        {
          email: this.formData.emailAddress,
          name: this.formData.firstName+' '+this.formData.lastName,
        });
        qualified('showFormExperience', 'experience-1772126772950');
      }
      let pdfDownloadLink = document.createElement('a');
      pdfDownloadLink.href = `/pdfs/${this.thisPage.meta.whitepaperFilename}`;
      pdfDownloadLink.download = this.thisPage.meta.whitepaperFilename;
      pdfDownloadLink.click();
      this.cloudSuccess = true;
    },
    handleScrollingInArticle: function () {
      let rightNavBar = document.querySelector('div[purpose="right-sidebar"]');
      let scrollTop = window.pageYOffset;
      let windowHeight = window.innerHeight;
      // Add/remove the 'header-hidden' class to the right sidebar to scroll it upwards with the website's header.
      if (rightNavBar) {
        if (scrollTop > this.scrollDistance && scrollTop > windowHeight * 1.5) {
          rightNavBar.classList.add('header-hidden');
          this.lastScrollTop = scrollTop;
        } else if(scrollTop < this.lastScrollTop - 60) {
          rightNavBar.classList.remove('header-hidden');
          this.lastScrollTop = scrollTop;
        }
      }
      this.scrollDistance = scrollTop;
    },
  }
});
