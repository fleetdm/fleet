parasails.registerPage('documentation', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    inputTextValue: '',
    inputTimers: {},
    searchString: '',
    showDocsNav: false,
    tree: [
      {
        title: 'Using Fleet',
        children: [
          'Fleet UI',
          'fleetctl',
          'REST API',
          'Osquery logs',
          'Monitoring Fleet',
          'Security best practices',
          'Updating Fleet',
          'FAQ - Using Fleet'
        ]
      },
      {
        title: 'Deploying',
        children: [
          'Installation',
          'Configuration',
          'Adding hosts',
          'Osquery logs',
          'Example deployment scenarios',
          'Self-managed agent updates',
          'FAQ - Deploying'
        ]
      },
      {
        title: 'Contributing',
        children: [
          'Building Fleet',
          'Testing',
          'Migrations',
          'Committing changes',
          'Releasing Fleet',
          'FAQ - Contributing'
        ]
      }
    ]
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    toggleDocsNav: function () {
      this.showDocsNav = !this.showDocsNav;
    },

    clickCTA: function (slug) {
      window.location = slug;
    },

    delayInput: function (callback, ms, label) {
      let inputTimers = this.inputTimers;
      return function () {
        label = label || 'defaultTimer';
        _.has(inputTimers, label) ? clearTimeout(inputTimers[label]) : 0;
        inputTimers[label] = setTimeout(callback, ms);
      };
    },

    setSearchString: function () {
      this.searchString = this.inputTextValue;
    },

    getSubtopics: function () {
      return this.body.filter((item) => item.type === 'subtopic')
        .map((item) => item.content);
    },

    getRelatedTopics: function () {
      try {
        const sectionIndex = this.outline.sections.findIndex((section) => section.title === this.currentpage.section);
        const topicIndex = this.outline.sections[sectionIndex].topics.findIndex((topic) => topic.title === this.currentpage.topic);
        return this.outline.sections[sectionIndex].topics[topicIndex].relatedTopics;
      } catch (error) {
        console.log(error);
        return [];
      }
    },

  }
});
