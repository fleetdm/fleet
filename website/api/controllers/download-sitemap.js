module.exports = {


  friendlyName: 'Download sitemap',


  description: 'Download sitemap file (returning a stream).',


  extendedDescription: `Notes:
  • Sitemap building inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L112-L180 and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js
  • Why escape XML?  See http://stackoverflow.com/questions/3431280/validation-problem-entityref-expecting-what-should-i-do and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L161-L172
  `,


  exits: {
    success: { outputFriendlyName: 'Sitemap (XML)', outputType: 'string' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({}) {

    if (sails.config.environment === 'staging') {
      // This explicit check for staging allows for the sitemap to still be developed/tested locally,
      // and for the real thing to be served in production, while explicitly preventing the "whoops,
      // i deployed staging and search engine crawlers got fixated on the wrong sitemap" dilemma.
      throw new Error('Since this is the staging environment, prevented sitemap.xml from being served to avoid search engine accidents.');
    }

    if (!_.isObject(sails.config.builtStaticContent)) {
      throw {badConfig: 'builtStaticContent'};
    } else if (!_.isArray(sails.config.builtStaticContent.queries)) {
      throw {badConfig: 'builtStaticContent.queries'};
    } else if (!_.isArray(sails.config.builtStaticContent.markdownPages)) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    // Start with sitemap.xml preamble + the root relative URLs of other webpages that aren't being generated from markdown
    let sitemapXml = '<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">';
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    //  ╦ ╦╔═╗╔╗╔╔╦╗   ╔═╗╔═╗╔╦╗╔═╗╔╦╗  ╔═╗╔═╗╔═╗╔═╗╔═╗
    //  ╠═╣╠═╣║║║ ║║───║  ║ ║ ║║║╣  ║║  ╠═╝╠═╣║ ╦║╣ ╚═╗
    //  ╩ ╩╩ ╩╝╚╝═╩╝   ╚═╝╚═╝═╩╝╚═╝═╩╝  ╩  ╩ ╩╚═╝╚═╝╚═╝
    let HAND_CODED_HTML_PAGES = [
      '/',
      '/fleetctl-preview',
      '/company/contact',
      '/queries',
      '/pricing',
      '/transparency',
      '/docs',
      '/logos',
      '/reports/state-of-device-management',
      '/releases',
      '/success-stories',
      '/securing',
      '/engineering',
      '/guides',
      '/announcements',
      '/report',
      '/deploy',
      '/podcasts',
      '/device-management',
      '/support',
      // FUTURE: Do something smarter to get hand-coded HTML pages from routes.js, like how rebuild-cloud-sdk works, to avoid this manual duplication.
      // See also https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js#L27
    ];
    for (let url of HAND_CODED_HTML_PAGES) {
      let trimmedRootRelativeUrl = _.trimRight(url,'/');// « really only necessary for home page; run on everything as a failsafe against accidental dupes due to trailing slashes in the list above
      sitemapXml += `<url><loc>${_.escape(sails.config.custom.baseUrl+trimmedRootRelativeUrl)}</loc></url>`;
    }//∞
    //  ╔╦╗╦ ╦╔╗╔╔═╗╔╦╗╦╔═╗  ╔═╗╔═╗╦═╗   ╔═╗ ╦ ╦╔═╗╦═╗╦ ╦  ╔═╗╔═╗╔═╗╔═╗╔═╗
    //   ║║╚╦╝║║║╠═╣║║║║║    ╠═╝║╣ ╠╦╝───║═╬╗║ ║║╣ ╠╦╝╚╦╝  ╠═╝╠═╣║ ╦║╣ ╚═╗
    //  ═╩╝ ╩ ╝╚╝╩ ╩╩ ╩╩╚═╝  ╩  ╚═╝╩╚═   ╚═╝╚╚═╝╚═╝╩╚═ ╩   ╩  ╩ ╩╚═╝╚═╝╚═╝
    for (let query of sails.config.builtStaticContent.queries) {
      sitemapXml +=`<url><loc>${_.escape(sails.config.custom.baseUrl+`/queries/${query.slug}`)}</loc></url>`;// note we omit lastmod for some sitemap entries. This is ok, to mix w/ other entries that do have lastmod. Why? See https://docs.google.com/document/d/1SbpSlyZVXWXVA_xRTaYbgs3750jn252oXyMFLEQxMeU/edit
    }//∞
    //  ╔╦╗╦ ╦╔╗╔╔═╗╔╦╗╦╔═╗  ╔═╗╔═╗╔═╗╔═╗╔═╗  ╔═╗╦═╗╔═╗╔╦╗  ╔╦╗╔═╗╦═╗╦╔═╔╦╗╔═╗╦ ╦╔╗╔
    //   ║║╚╦╝║║║╠═╣║║║║║    ╠═╝╠═╣║ ╦║╣ ╚═╗  ╠╣ ╠╦╝║ ║║║║  ║║║╠═╣╠╦╝╠╩╗ ║║║ ║║║║║║║
    //  ═╩╝ ╩ ╝╚╝╩ ╩╩ ╩╩╚═╝  ╩  ╩ ╩╚═╝╚═╝╚═╝  ╚  ╩╚═╚═╝╩ ╩  ╩ ╩╩ ╩╩╚═╩ ╩═╩╝╚═╝╚╩╝╝╚╝
    for (let pageInfo of sails.config.builtStaticContent.markdownPages) {
      sitemapXml +=`<url><loc>${_.escape(sails.config.custom.baseUrl+pageInfo.url)}</loc><lastmod>${_.escape(new Date(pageInfo.lastModifiedAt).toJSON())}</lastmod></url>`;
    }//∞
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    sitemapXml += '</urlset>';

    // Set MIME type for content-type response header.
    this.res.type('text/xml');

    // Respond with XML.
    return sitemapXml;
  }


};
