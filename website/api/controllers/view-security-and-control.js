module.exports = {


  friendlyName: 'View security and control',


  description: 'Display "Security and control" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/security-and-control'
    }

  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      // Specify an order for the testimonials on this page using the last names of quote authors
      // Note: this page uses the same testimonials as the /software-managment page.
      let testimonialOrderForThisPage = [
        'Luis Madrigal',
        'Arsenio Figueroa',
        'Bart Reardon',
        'Andre Shields',
        'Wes Whetstone',
        'Nico Waisman',
        'Chandra Majumdar',
        'Kenny Botelho',
        'Erik Gomez',
        'Eric Tan',
        'Adam Pippert',
        'Justin LaBo',
        'Brian LaShomb',
        'Ed Merrett',
      ];

      // Filter the testimonials by product category
      testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
        return _.contains(testimonial.productCategories, 'Software management') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
      });

      testimonialsForScrollableTweets.sort((a, b)=>{
        if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
          return 1;
        } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
          return -1;
        }
        return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
      });
    }//ﬁ

    // Build a JSON FAQ to insert into this page's header.
    let pageFaqForSeo = {
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      'mainEntity': [
        {
          '@type': 'Question',
          'name': 'What can Fleet detect on a device?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet inventories more than installed applications. It reports on browser extensions across Chrome, Firefox, Safari, and Edge, IDE plugins, AI tools and MCP connectors, package manager installs like Homebrew and NPM, Python libraries, and operating system versions, across macOS, Windows, Linux, and more.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can Fleet see browser extensions across my fleet?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. Fleet reports every extension installed across Chrome, Firefox, Safari, and Edge, so you can find extensions that haven\'t been vetted or approved.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet block or remove unauthorized software?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'You set policies for what software is approved across your fleet. Fleet can block, remove, or allowlist software automatically. When it detects something unauthorized, Fleet can remove it or notify the user, depending on how your team prefers to work.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet prioritize vulnerabilities?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Not every CVE is equal. Fleet uses CISA Known Exploited Vulnerabilities (KEV) and EPSS data so your team focuses on vulnerabilities that are actually being exploited, not just those with high CVSS scores.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Does Fleet require network vulnerability scans?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'No. Fleet uses a lightweight agent instead of network scans, so you get up-to-date data from on-premises and remote devices without clogging your network.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can Fleet remediate vulnerabilities automatically?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. When a device has vulnerable software, Fleet can install the correct version automatically, with no ticket and no manual intervention. When an update isn\'t available, Fleet can uninstall old versions, apply a workaround, or remove stale apps.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Which operating systems does Fleet support?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet manages and reports across macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS from one place, so you spend less time reconciling data from multiple tools.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet track AI tools and MCP connectors?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet reports on which AI tools, MCP connectors, and IDE extensions are in use across your organization, so you can track adoption and identify unsanctioned tools.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet patch software without disrupting users?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet installs updates when apps are idle and notifies users before a reboot. You can exempt specific devices or people when needed.'
          }
        }
      ]
    };


    // Respond with view.
    return {
      testimonialsForScrollableTweets,
      pageFaqForSeo: JSON.stringify(pageFaqForSeo),
    };


  }


};
