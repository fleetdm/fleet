module.exports = {


  friendlyName: 'View compare',


  description: 'Display "Compare" page, a hub that links to Fleet\'s comparison articles.',


  exits: {

    success: {
      viewTemplatePath: 'pages/compare'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages)) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    // Card copy comes from the /compare design in Figma. A card only shows once its comparison article is published.
    let headToHeadComparisons = [
      { slug: 'jamf-vs-fleet', title: 'Fleet vs. Jamf', description: 'Compare Fleet and Jamf Pro on platform support, enrollment, software management, security, reporting, and pricing.' },
      { slug: 'intune-vs-fleet', title: 'Fleet vs. Intune', description: 'Compare Fleet and Microsoft Intune across Windows, macOS, and Linux: compliance checks, vulnerability detection, reporting speed, patching, self-hosting, and pricing.' },
      { slug: 'iru-vs-fleet', title: 'Fleet vs. Iru (formerly Kandji)', description: 'Compare Fleet and Iru on platform support, patching, security, self-hosting, pricing, and support.' },
      { slug: 'tanium-vs-fleet', title: 'Fleet vs. Tanium', description: 'Compare Fleet and Tanium on platform support, live queries, MDM, patching, security, deployment, and pricing.' },
      { slug: 'workspace-one-vs-fleet', title: 'Fleet vs. Workspace ONE', description: 'Compare Fleet and Workspace ONE on multi-platform, deployment, and security capabilities.' },
    ];
    let multiVendorComparisons = [
      { slug: 'jamf-vs-intune-vs-fleet', title: 'Fleet vs. Jamf vs. Intune', description: 'Compare how Fleet, Jamf Pro, and Microsoft Intune manage Apple devices: enrollment, configuration, software and patching, and security.' },
      { slug: 'jamf-vs-iru-vs-fleet', title: 'Fleet vs. Jamf vs. Iru (formerly Kandji)', description: 'Compare Fleet, Jamf, and Iru for platform support, pricing, GitOps, security, and API capabilities to find the right MDM for your team.' },
      { slug: 'jumpcloud-vs-workspace-one-vs-fleet', title: 'Fleet vs. JumpCloud vs. Workspace ONE', description: 'Compare Fleet, JumpCloud, and Workspace ONE as Jamf alternatives for managing macOS, Windows, Linux, and mobile devices.' },
    ];

    let publishedUrls = _.pluck(_.filter(sails.config.builtStaticContent.markdownPages, (page)=>{
      return _.startsWith(page.url, '/compare/') && page.meta && page.meta.category === 'comparison';
    }), 'url');
    let withUrlIfPublished = (comparisons)=>{
      return _.filter(_.map(comparisons, (comparison)=>{
        return _.extend({}, comparison, { url: '/compare/'+comparison.slug });
      }), (comparison)=>{
        return _.contains(publishedUrls, comparison.url);
      });
    };

    return {
      headToHeadComparisons: withUrlIfPublished(headToHeadComparisons),
      multiVendorComparisons: withUrlIfPublished(multiVendorComparisons),
    };

  }


};
