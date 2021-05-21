module.exports = {


  friendlyName: 'Download sitemap',


  description: 'Download sitemap file (returning a stream).',


  exits: {
    success: { outputFriendlyName: 'Sitemap (sitemap.xml)' }
  },


  fn: async function ({}) {

    if (sails.config.environment === 'staging') {
      // This explicit check for staging allows for the sitemap to still be developed/tested locally,
      // and for the real thing to be served in production, while explicitly preventing the "whoops,
      // i deployed staging and search engine crawlers got fixated on the wrong sitemap" dilemma.
      throw new Error('Since this is the staging environment, prevented sitemap.xml from being served to avoid search engine accidents.');
    }//•

    // Notes:
    // • sitemap building inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L112-L180 and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js
    // • Why escape XML?  See http://stackoverflow.com/questions/3431280/validation-problem-entityref-expecting-what-should-i-do and https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/refresh.js#L161-L172

    // Start with sitemap.xml preamble + the root relative URLs of other webpages that aren't being generated from markdown
    let sitemapXml = '<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">';
    let HAND_CODED_HTML_PAGES = [
      '/',
      '/get-started',
      // TODO rest  (e.g. hand-coded HTML pages from routes.js -- see https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/helpers/get-pages-for-sitemap.js#L27)
    ];
    for (let url of HAND_CODED_HTML_PAGES) {
      let trimmedRootRelativeUrl = _.trimRight(url,'/');
      sitemapXml += `<url><loc>${_.escape(sails.config.custom.baseUrl+trimmedRootRelativeUrl)}</loc></url>`;// note we omit lastmod. This is ok, to mix w/ other entries that do have lastmod. Why? See https://docs.google.com/document/d/1SbpSlyZVXWXVA_xRTaYbgs3750jn252oXyMFLEQxMeU/edit
    }//∞
    for (let pageInfo of sails.config.builtStaticContent.allPages) {
      let trimmedRootRelativeUrl = _.trimRight(pageInfo.url,'/');
      sitemapXml +=`<url><loc>${_.escape(sails.config.custom.baseUrl+trimmedRootRelativeUrl)}</loc><lastmod>${_.escape(new Date(pageInfo.lastModifiedAt).toJSON())}</lastmod></url>`;
    }//∞
    sitemapXml += '</urlset>';

    // Set MIME type for content-type response header.
    this.res.type('text/xml');

    // Respond with XML.
    return sitemapXml;
  }


};
