parasails.registerPage('workshops', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    slideCount: 6,// number of unique slides in the carousel
    position: 6,// The starting position of the carousel indicator
    animate: true,// Toggled off for the silent (no-transition) snap-back.
    sliding: false,// set to prevent clicks while the carousel is mid-transition.
    selectedType: '',
    citySearch: '',
    workshopsPerPage: 12,
    numberOfWorkshopsShown: 12,
    iconByType: {
      'Apple administrator workshop': '/images/logo-apple-36x36@2x.png',
      'Windows administrator workshop': '/images/logo-windows-36x36@2x.png',
      'Linux administrator workshop': '/images/logo-linux-36x36@2x.png',
      'GitOps workshop': '/images/icon-ai-sparkles-32x32.svg',
    },
    trackByType: {
      'Apple administrator workshop': 'apple',
      'Windows administrator workshop': 'windows',
      'Linux administrator workshop': 'linux',
      'GitOps workshop': 'ai',
    },
    heroPlatform: 'apple',// Set from the visitor's OS in beforeMount.
    heroExamples: {
      apple: {
        prompt: 'Turn on FileVault for every Mac in Workstations and lock screens after 10 minutes.',
        settingsKey: 'apple_settings',
        addedLines: ['enable_disk_encryption: true', 'configuration_profiles:', '  - path: ../lib/screen-lock.mobileconfig'],
        appliedTo: 'Applied to every Mac in Workstations',
      },
      windows: {
        prompt: 'Turn on BitLocker for every Windows PC in Workstations and require a startup PIN.',
        settingsKey: 'windows_settings',
        addedLines: ['enable_disk_encryption: true', 'require_bitlocker_pin: true'],
        appliedTo: 'Applied to every Windows PC in Workstations',
      },
      linux: {
        prompt: 'Escrow the disk encryption key for every Linux laptop in Workstations.',
        settingsKey: 'linux_settings',
        addedLines: ['enable_escrow_disk_encryption_key: true'],
        appliedTo: 'Applied to every Linux laptop in Workstations',
      },
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // macOS, iOS, iPadOS, and anything we can't identify keep the Apple example.
    if (typeof bowser === 'undefined') {
      return;
    }
    if (bowser.windows) {
      this.heroPlatform = 'windows';
    } else if (bowser.linux) {
      this.heroPlatform = 'linux';
    }
  },

  //  ╔═╗╔═╗╔╦╗╔═╗╦ ╦╔╦╗╔═╗╔╦╗
  //  ║  ║ ║║║║╠═╝║ ║ ║ ║╣  ║║
  //  ╚═╝╚═╝╩ ╩╩  ╚═╝ ╩ ╚═╝═╩╝
  computed: {
    heroExample: function() {
      return this.heroExamples[this.heroPlatform];
    },
    heroDiffLines: function() {
      let lines = [
        { text: '  controls:', added: false },
        { text: '    ' + this.heroExample.settingsKey + ':', added: false },
      ];
      for (let line of this.heroExample.addedLines) {
        lines.push({ text: '+     ' + line, added: true });
      }
      return lines;
    },
    activeDot: function() {
      // Controls what carousel indicator is highlighted, regardless of which copy we're in.
      return ((this.position % this.slideCount) + this.slideCount) % this.slideCount;
    },
    countByType: function() {
      return _.countBy(this.futureWorkshops, 'type');
    },
    workshopTypes: function() {
      return _.uniq(_.pluck(this.futureWorkshops, 'type'));
    },
    filteredWorkshops: function() {
      let search = this.citySearch.trim().toLowerCase();
      return _.filter(this.futureWorkshops, (event)=>{
        if (this.selectedType && event.type !== this.selectedType) {
          return false;
        }
        return !search || _.contains((event.workshopCity || '').toLowerCase(), search);
      });
    },
    workshopsByMonth: function() {
      let months = [];
      let monthsByLabel = {};
      _.each(this.filteredWorkshops, (event, index)=>{
        // eventTime is formatted by the view action in the event's own time zone, e.g. "Oct 20 from 1pm to 5pm MDT".
        let [date, timeRange] = event.eventTime.split(' from ');
        let [monthAbbreviation, day] = date.split(' ');
        // Workshops cached before startYear existed fall back to the UTC year.
        let label = monthAbbreviation + ' ' + (event.startYear || new Date(event.startsAt).getUTCFullYear());
        if (!monthsByLabel[label]) {
          monthsByLabel[label] = { label, totalCount: 0, workshops: [] };
          months.push(monthsByLabel[label]);
        }
        monthsByLabel[label].totalCount++;
        if (index < this.numberOfWorkshopsShown) {
          monthsByLabel[label].workshops.push(Object.assign({}, event, { month: monthAbbreviation, day, timeRange }));
        }
      });
      return _.filter(months, (month)=>{ return month.workshops.length > 0; });
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

    clickScrollTo: function(elementId) {
      document.getElementById(elementId).scrollIntoView({ behavior: 'smooth' });
    },

    clickSelectType: function(type) {
      this.selectedType = type;
      this.numberOfWorkshopsShown = this.workshopsPerPage;
    },

    clickFilterAndScroll: function(type) {
      this.clickSelectType(type);
      this.citySearch = '';
      this.clickScrollTo('upcoming');
    },

    clickShowMoreWorkshops: function() {
      this.numberOfWorkshopsShown += this.workshopsPerPage;
    },
  }
});
