module.exports = {


  friendlyName: 'Download one rss feed',


  description: 'Download one rss feed file (returning a stream).',


  inputs: {

    category: {
      type: 'string',
      required: true,
    }

  },


  exits: {
    success: { outputFriendlyName: 'RSS feed', outputType: 'string' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({category}) {

    if (!_.isObject(sails.config.builtStaticContent)) {
      throw {badConfig: 'builtStaticContent'};
    } else if (!_.isArray(sails.config.builtStaticContent.markdownPages)) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }


    let articlesToAddToFeed = [];
    if (category === 'articles') {
      // If the category is `articles` we'll build a rss feed that contains all articles
      articlesToAddToFeed = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.htmlId, 'articles')) {
          return page;
        }
      });
    } else {
      // If the user requested a specific category, we'll only build a feed with articles in that category
      articlesToAddToFeed = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.url, '/'+category)) {
          return page;
        }
      });
    }
    let articleCategory = '';
    let categoryDescription = '';
    // Set a description and title for this RSS feed.
    switch(category) {
      case 'device-management':
        articleCategory = 'Success stories';
        categoryDescription = 'Read about how others are using Fleet and osquery.';
        break;
      case 'securing':
        articleCategory = 'Security';
        categoryDescription = 'Learn more about how we secure Fleet.';
        break;
      case 'releases':
        articleCategory = 'Releases';
        categoryDescription = 'Read about the latest release of Fleet.';
        break;
      case 'engineering':
        articleCategory = 'Engineering';
        categoryDescription = 'Read about engineering at Fleet and beyond.';
        break;
      case 'guides':
        articleCategory = 'Guides';
        categoryDescription = 'Learn more about how to use Fleet to accomplish your goals.';
        break;
      case 'announcements':
        articleCategory = 'Announcements';
        categoryDescription = 'The latest news from Fleet.';
        break;
      case 'deploy':
        articleCategory = 'Deployment guides';
        categoryDescription = 'Learn more about how to deploy Fleet.';
        break;
      case 'podcasts':
        articleCategory = 'Podcasts';
        categoryDescription = 'Listen to the Future of Device Management podcast';
        break;
      case 'report':
        articleCategory = 'Reports';
        categoryDescription = '';
        break;
      case 'articles':
        articleCategory = 'Fleet for osquery';
        categoryDescription = 'Read all articles from Fleet\'s blog.';
    }

    // Start building the rss feed
    let rssFeedXml = '<rss version="2.0"><channel>';

    let rssFeedTitle = `<title>Fleet blog | ${_.escape(articleCategory)}</title>`;
    let rssFeedDescription = `<description>${_.escape(categoryDescription)}</description>`;
    let rsslastBuildDate = `<lastBuildDate>${_.escape(new Date(Date.now()))}</lastBuildDate>`;
    let rssFeedImage = `<image><link>${_.escape('https://fleetdm.com'+category)}</link><title>${_.escape('Fleet Blog | '+articleCategory)}</title><url>${_.escape('https://fleetdm.com/images/fleet-logo-square@2x.png')}</url></image>`;

    rssFeedXml += `${rssFeedTitle}${rssFeedDescription}${rsslastBuildDate}${rssFeedImage}`;



    for (let pageInfo of articlesToAddToFeed) {
      let rssItemTitle = `<title>${_.escape(pageInfo.meta.articleTitle)}</title>`;
      let rssItemDescription = `<description>${_.escape(pageInfo.meta.description)}</description>`;
      let rssItemLink = `<link>${_.escape('https://fleetdm.com'+pageInfo.url)}</link>`;
      let rssItemPublishDate = `<pubDate>${_.escape(new Date(pageInfo.meta.publishedOn).toJSON())}</pubDate>`
      let rssItemImage = '';
      if(pageInfo.meta.articleImageUrl){
        rssItemImage = `<image><link>${_.escape('https://fleetdm.com'+pageInfo.url)}</link><title>${_.escape(pageInfo.meta.articleTitle)}</title><url>${_.escape('https://fleetdm.com'+pageInfo.meta.articleImageUrl)}</url></image>`
      }
      // Add the article to the feed.
      rssFeedXml += `<item>${rssItemTitle}${rssItemDescription}${rssItemLink}${rssItemImage}${rssItemPublishDate}</item>`
    }

    rssFeedXml += `</channel></rss>`;

    this.res.type('text/xml');

    // All done.
    return rssFeedXml;

  }


};
