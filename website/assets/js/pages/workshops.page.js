parasails.registerPage('workshops', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    slideCount: 4,// number of unique slides in the carousel
    position: 4,// The starting position of the carousel indicator
    animate: true,// Toggled off for the silent (no-transition) snap-back.
    sliding: false,// set to prevent clicks while the carousel is mid-transition.
  },

  //  ╔═╗╔═╗╔╦╗╔═╗╦ ╦╔╦╗╔═╗╔╦╗
  //  ║  ║ ║║║║╠═╝║ ║ ║ ║╣  ║║
  //  ╚═╝╚═╝╩ ╩╩  ╚═╝ ╩ ╚═╝═╩╝
  computed: {
    activeDot: function() {
      // Controls what carousel indicator is highlighted, regardless of which copy we're in.
      return ((this.position % this.slideCount) + this.slideCount) % this.slideCount;
    },
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickCarouselIndicator: function(direction) {
      if (this.sliding) { return; }
      this.sliding = true;
      if(direction === 'previous') {
        this.position -= 1;
      } else if (direction === 'next') {
        this.position += 1;

      }
    },

    clickGoToSlide: function(dotIndex) {
      if (this.sliding || this.activeDot === dotIndex) { return; }
      this.sliding = true;
      this.position = this.slideCount + dotIndex;// Jump within the middle copy.
    },

    handleTransitionEnd: function() {
      this.sliding = false;
      // If a slide left the middle copy, snap back a copy-width (no animation) so
      // we can keep going forever with photos still peeking on both sides.
      if (this.position < this.slideCount || this.position >= this.slideCount * 2) {
        this.animate = false;
        this.position = (this.slideCount + this.activeDot);
        this.$nextTick(() => {
          window.requestAnimationFrame(() => {
            window.requestAnimationFrame(() => {
              this.animate = true;
            });
          });
        });
      }
    },
  }
});
