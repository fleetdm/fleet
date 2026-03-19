module.exports = {


  friendlyName: 'Download rss feed',


  description: 'Generate and return an RSS feed for a category of Fleet\'s articles',


  inputs: {

    categoryName: {
      type: 'string',
      required: true,
      isIn: [
        'success-stories',
        'securing',
        'releases',
        'engineering',
        'guides',
        'announcements',
        'deploy',
        'podcasts',
        'report',
        'articles',
      ],
    }

  },


  exits: {
    success: { outputFriendlyName: 'RSS feed XML', outputType: 'string' },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function ({categoryName}) {

    if (!_.isObject(sails.config.builtStaticContent)) {
      throw {badConfig: 'builtStaticContent'};
    } else if (!_.isArray(sails.config.builtStaticContent.markdownPages)) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    // Build the feed URL and channel link for this category.
    let feedUrl = `https://fleetdm.com/rss/${categoryName}`;
    let channelLink = categoryName === 'articles' ? 'https://fleetdm.com/articles' : `https://fleetdm.com/${categoryName}`;

    // Start building the rss feed with XML declaration and Atom namespace.
    let rssFeedXml = `<?xml version="1.0" encoding="UTF-8"?>`;
    rssFeedXml += `<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel>`;

    // Build the description and title for this RSS feed.
    let articleCategoryTitle = '';
    let categoryDescription = '';
    switch(categoryName) {
      case 'success-stories':
        articleCategoryTitle = 'Success stories | Fleet blog';
        categoryDescription = 'Read about how others are using Fleet and osquery.';
        break;
      case 'securing':
        articleCategoryTitle = 'Security | Fleet blog';
        categoryDescription = 'Learn more about how we secure Fleet.';
        break;
      case 'releases':
        articleCategoryTitle = 'Releases | Fleet blog';
        categoryDescription = 'Read about the latest release of Fleet.';
        break;
      case 'engineering':
        articleCategoryTitle = 'Engineering | Fleet blog';
        categoryDescription = 'Read about engineering at Fleet and beyond.';
        break;
      case 'guides':
        articleCategoryTitle = 'Guides | Fleet blog';
        categoryDescription = 'Learn more about how to use Fleet to accomplish your goals.';
        break;
      case 'announcements':
        articleCategoryTitle = 'Announcements | Fleet blog';
        categoryDescription = 'The latest news from Fleet.';
        break;
      case 'deploy':
        articleCategoryTitle = 'Deployment guides | Fleet blog';
        categoryDescription = 'Learn more about how to deploy Fleet.';
        break;
      case 'podcasts':
        articleCategoryTitle = 'Podcasts | Fleet blog';
        categoryDescription = 'Listen to the Future of Device Management podcast';
        break;
      case 'report':
        articleCategoryTitle = 'Reports | Fleet blog';
        categoryDescription = '';
        break;
      case 'articles':
        articleCategoryTitle = 'Fleet blog | Fleet';
        categoryDescription = 'Read all articles from Fleet\'s blog.';
    }

    let rssFeedTitle = `<title>${_.escape(articleCategoryTitle)}</title>`;
    let rssFeedDescription = `<description>${_.escape(categoryDescription)}</description>`;
    let rssFeedLink = `<link>${_.escape(channelLink)}</link>`;
    let rssAtomSelfLink = `<atom:link href="${_.escape(feedUrl)}" rel="self" type="application/rss+xml"/>`;
    let rsslastBuildDate = `<lastBuildDate>${new Date().toUTCString()}</lastBuildDate>`;
    let rssFeedLanguage = `<language>en</language>`;
    let rssFeedDocs = `<docs>https://www.rssboard.org/rss-specification</docs>`;
    let rssFeedImage = `<image><link>${_.escape(channelLink)}</link><title>${_.escape(articleCategoryTitle)}</title><url>${_.escape('https://fleetdm.com/images/fleet-logo-square@2x.png')}</url></image>`;

    rssFeedXml += `${rssFeedTitle}${rssFeedDescription}${rssFeedLink}${rssAtomSelfLink}${rsslastBuildDate}${rssFeedLanguage}${rssFeedDocs}${rssFeedImage}`;


    // Determine the subset of articles that will be used to squirt out an XML string.
    let articlesToAddToFeed = [];
    if (categoryName === 'articles') {
      // If the category is `articles` we'll build a rss feed that contains all articles
      articlesToAddToFeed = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.htmlId, 'articles')) {
          return page;
        }
      });//∞
    } else {
      // If the user requested a specific category, we'll only build a feed with articles in that category
      articlesToAddToFeed = sails.config.builtStaticContent.markdownPages.filter((page)=>{
        if(_.startsWith(page.url, '/'+categoryName)) {
          return page;
        }
      });//∞
    }

    // Iterate through the filtered array of articles, adding <item> elements for each article.
    for (let pageInfo of articlesToAddToFeed) {
      let itemLink = `https://fleetdm.com${pageInfo.url}`;
      let rssItemTitle = `<title>${_.escape(pageInfo.meta.articleTitle)}</title>`;
      let rssItemDescription = `<description>${_.escape(pageInfo.meta.description)}</description>`;
      let rssItemLink = `<link>${_.escape(itemLink)}</link>`;
      let rssItemGuid = `<guid isPermaLink="true">${_.escape(itemLink)}</guid>`;
      let rssItemPublishDate = `<pubDate>${new Date(pageInfo.meta.publishedOn).toUTCString()}</pubDate>`;
      // Add the article to the feed.
      rssFeedXml += `<item>${rssItemTitle}${rssItemDescription}${rssItemLink}${rssItemGuid}${rssItemPublishDate}</item>`;
    }

    rssFeedXml += `</channel></rss>`;

    // Set the response type
    this.res.type('text/xml');

    // Return the generated RSS feed
    return rssFeedXml;

  }


};
