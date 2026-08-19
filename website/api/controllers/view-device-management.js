module.exports = {


  friendlyName: 'View device management',


  description: 'Display "Device management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/device-management'
    },
    badConfig: { responseType: 'badConfig' },
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
      let testimonialOrderForThisPage = [
        'Bart Reardon',
        'Scott MacVicar',
        'Mike Meyer',
        'Luis Madrigal',
        'Tom Larkin',
        'Kenny Botelho',
        'Erik Gomez',
        'Chandra Majumdar',
        'Eric Tan',
        'matt carr',
        'Nico Waisman',
        'Adam Pippert',
        'Philip Chotipradit',
        'Roger Cantrell',
        'Chayce O\'Neal',
        'David Bodmer',
        'Fiona Skelton',
      ];

      // Filter the testimonials by product category
      testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
        return _.contains(testimonial.productCategories, 'Device management') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
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
          'name': 'Which operating systems can Fleet manage?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet lets you configure, update, and secure macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS devices with one tool, so you can replace fragmented point solutions with a single platform your whole team can use.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How quickly can I get answers about my devices?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet maintains a live connection to every device, so you get answers in seconds, not hours. You get near-instant visibility into your Apple, Linux, Windows, and BYOD devices from a single, open platform.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet patch and update software?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Deploy software from a built-in catalog of Fleet-maintained apps or upload your own packages. Fleet checks every device continuously and installs updates automatically when something falls out of compliance, with no tickets and no manual intervention.'
          }
        },
        {
          '@type': 'Question',
          'name': 'What are Fleet-maintained apps?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet-maintained apps let you install common software with no extra configuration. Fleet handles packaging, versioning, and updates for you.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can end users install software themselves?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. End users can install approved software themselves, without filing a ticket. Self-service is available on macOS, Windows, and Linux.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can Fleet enforce OS updates?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. You can set minimum OS versions and deadlines across macOS, Windows, iOS, and iPadOS.'
          }
        },
        {
          '@type': 'Question',
          'name': 'How does Fleet help with compliance?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Fleet enforces security policies and demonstrates compliance across your entire fleet with live results, not stale reports from last week\'s scan. You can run live checks against CIS benchmarks, SOC 2, ISO, or your own baselines and see results in seconds.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Does Fleet support disk encryption?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. Fleet can enforce FileVault, BitLocker, and LUKS2 across macOS, Windows, and Linux.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can Fleet remotely lock or wipe devices?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. You can lock or wipe devices remotely across macOS, Windows, and Linux.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Can I run Fleet alongside my existing MDM?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Yes. You can deploy Fleet alongside an existing MDM to see device states across organizations, without disrupting employees.'
          }
        },
        {
          '@type': 'Question',
          'name': 'Where can Fleet be deployed?',
          'acceptedAnswer': {
            '@type': 'Answer',
            'text': 'Run Fleet on-prem, in the cloud, or air-gapped, with no lock-in and no black boxes. You can change deployment models without rebuilding your stack, and keep control of your infrastructure and your data.'
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
