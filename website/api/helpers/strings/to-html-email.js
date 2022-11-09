module.exports = {


  friendlyName: 'To HTML email',


  description: 'Compile a Markdown string into an HTML string with styles added inline for the Fleet newsletter.',


  extendedDescription:
    'Expects GitHub-flavored Markdown syntax.  Uses [`marked`](https://github.com/chjj/marked)@v0.3.5.  '+
    'Inspired by https://github.com/mikermcneil/machinepack-markdown/tree/5d8cee127e8ce45c702ec9bbb2b4f9bc4b7fafac',


  moreInfoUrl: 'https://help.github.com/articles/basic-writing-and-formatting-syntax/',


  sideEffects: 'cacheable',


  inputs: {

    mdString: {
      description: 'Markdown string to convert',
      example: '# hello world\n it\'s me, some markdown string \n\n ```js\n//but maybe i have code snippets too...\n```',
      required: true
    },
  },


  exits: {

    success: {
      outputFriendlyName: 'HTML',
      outputExample: '<h1 id="hello-world">hello world</h1>\n<p> it&#39;s me, some markdown string </p>\n<pre><code class="lang-js">//but maybe i have code snippets too...</code></pre>\n'
    },

    unsafeMarkdown: {
      friendlyName: 'Unsafe Markdown detected',
      description: 'The provided input contained unsafe content (like HTML tags).'
    }

  },


  fn: function(inputs, exits) {
    const { marked } = require('marked');


    var markedOpts = {
      gfm: true,
      tables: true,
      breaks: false,
      pedantic: false,
      smartLists: true,
      smartypants: false,
    };

    // Creating a custom renderer to add inline styles to HTML elements
    let customRenderer = new marked.Renderer();

    // For codeblocks
    customRenderer.code = function(codeHTML) {
      return '<pre style="padding: 24px; border: 1px solid #E2E4EA; overflow: auto; margin-bottom: 16px; margin-top: 16px; border-radius: 6px; background: #F9FAFC;"><code style="font-size: 13px; line-height: 16px; font-family: Source Code Pro;">'+_.escape(codeHTML)+'</code></pre>';
    };
    // For blockquotes
    customRenderer.blockquote = function(quoteHTML) {
      return `<blockquote>\n${quoteHTML}\n</blockquote>\n`;
    };

    customRenderer.heading = function(textHTML, level) {
      let inlineStyles;
      if(level === 1) { // For h1s
        inlineStyles = 'font-weight: 800; font-size: 24px; line-height: 32px; margin-bottom: 16px;';
      } else if (level === 2) { // For h2s
        inlineStyles = 'font-weight: 700; font-size: 20px; line-height: 28px; margin-bottom: 16px; margin-top: 32px;';
      } else if (level === 3) { // for h3s
        inlineStyles = 'font-weight: 700; font-size: 20px; line-height: 24px; margin-bottom: 16px;';
      } else {// H4s or higher
        inlineStyles = 'font-weight: 700; font-size: 16px; line-height: 20px; margin-bottom: 16px;';
      }
      return `<h${level} style="${inlineStyles}">\n${textHTML}\n</h${level}>\n`;
    };

    // For <hr> elements
    customRenderer.hr = function() {
      return `<hr style="border-top: 2px solid #E2E4EA; margin-top: 16px; margin-bottom: 16px; width: 100%;">`;
    };

    // For lists
    customRenderer.list = function(bodyHTML, ordered) {
      if(ordered){
        return `<ol style="padding-left: 16px; margin-bottom: 32px;">\n${bodyHTML}</ol>\n`;
      } else {
        return `<ul style="padding-left: 16px; margin-bottom: 32px;">\n${bodyHTML}</ul>\n`;
      }
    };

    // For list items
    customRenderer.listitem = function(textHTML) {
      return `<li style="margin-bottom: 16px;">\n${textHTML}\n</li>\n`;
    };

    customRenderer.paragraph = function(text) {
      return `<p style="font-size: 16px; line-height: 24px; font-weight: 400; margin-bottom: 16px;">\n${text}\n</p>\n`;
    };

    // For bold text
    customRenderer.strong = function(textHTML) {
      return `<strong style="display: inline; font-weight: 700; font-size: 16px; line-height: 24px;">${textHTML}</strong>`;
    };

    // For emphasized text
    customRenderer.em = function(textHTML) {
      return `<span style="display: inline; font-style: italic; font-size:16px;>${textHTML}</span>`;
    };

    // For inline codespans
    customRenderer.codespan = function(codeHTML) {
      return '<code style="display: inline; background: #F1F0FF; color: #192147; padding: 4px 8px; font-size: 13px; line-height: 16px; font-family: Courier New;">'+codeHTML+'</code>';
    };

    // For links
    customRenderer.link = function(href, title, textHTML) {
      (href)=>{
        let isExternal = ! href.match(/^https?:\/\/([^\.|blog]+\.)*fleetdm\.com/g);// « FUTURE: make this smarter with sails.config.baseUrl + _.escapeRegExp()
        // Check if this link is to fleetdm.com or www.fleetdm.com.
        let isBaseUrl = href.match(/^(https?:\/\/)([^\.]+\.)*fleetdm\.com$/g);
        if (isExternal) {
          href = href.replace(/(https?:\/\/([^"]+))/g, '$1 target="_blank"');
        } else {
          // Otherwise, change the link to be web root relative.
          // (e.g. 'href="http://sailsjs.com/documentation/concepts"'' becomes simply 'href="/documentation/concepts"'')
          // > Note: See the Git version history of "compile-markdown-content.js" in the sailsjs.com website repo for examples of ways this can work across versioned subdomains.
          if (isBaseUrl) {
            href = href.replace(/https?:\/\//, '');
          } else {
            href = href.replace(/https?:\/\//, '');
          }
        }
      };
      return `<a style="display: inline; color: #6A67FE; font-size: 16px; text-decoration: none; word-break: break-word;" href="${href}" target="_blank">${textHTML}</a>`;
    };

    // For images
    customRenderer.image = function(href, title) {
      let linkToImageInAssetsFolder = href.replace(/^(\.\.\/website\/assets)/gi, 'https://fleetdm.com');
      return `<img style="max-width: 100%; margin-top: 40px; margin-bottom: 40px;" src="${linkToImageInAssetsFolder}" alt="${title}">`;
    };

    markedOpts.renderer = customRenderer;

    // Now actually compile the markdown to HTML.
    marked(inputs.mdString, markedOpts, function afterwards (err, htmlString) {
      if (err) {
        return exits.error(err);
      }

      return exits.success(htmlString);
    });
  }


};
