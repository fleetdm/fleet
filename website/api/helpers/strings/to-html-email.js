module.exports = {


  friendlyName: 'To HTML',


  description: 'Compile a Markdown string into an HTML string.',


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

    allowHtml: {
      friendlyName: 'Allow HTML?',
      description: 'Whether or not to allow HTML tags in the Markdown input.  Defaults to `true`.',
      extendedDescription: 'If `false`, any input that contains HTML tags will trigger the `unsafeMarkdown` exit.',
      example: true,
      defaultsTo: true
    },

    addIdsToHeadings: {
      friendlyName: 'Add IDs to headings?',
      description: 'Whether or not to add an ID attribute to rendered heading tags like <h1>',
      extendedDescription: 'This is not part of the Markdown specification (see http://daringfireball.net/projects/markdown/dingus), but it is the default behavior for the `marked` module.  Defaults to `true`.',
      example: true,
      defaultsTo: false
    }

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

    // For full list of options, see:
    //  • https://github.com/chjj/marked
    var markedOpts = {
      gfm: true,
      tables: true,
      breaks: false,
      pedantic: false,
      smartLists: true,
      smartypants: false,
    };

    let customRenderer = new marked.Renderer();

    // if (inputs.addIdsToHeadings === true) {
    //   var headingsRenderedOnThisPage = [];
    //   customRenderer.heading = function (text, level) {
    //     // If the heading has underscores and no spaces (e.g. osquery_async_host_collect_log_stats_interval) we'll add optional linebreaks before each underscore
    //     var textWithLineBreaks;
    //     if(text.match(/\S(\w+\_\S)+(\w\S)+/g) && !text.match(/\s/g)){
    //       textWithLineBreaks = text.replace(/(\_)/g, '&#8203;_');
    //     }
    //     var headingID = _.kebabCase(_.unescape(text).replace(/[\’\']/g, ''));
    //     if(!_.contains(headingsRenderedOnThisPage, headingID)){
    //       headingsRenderedOnThisPage.push(headingID);
    //     } else {
    //       headingID = sails.helpers.strings.ensureUniq(headingID, headingsRenderedOnThisPage);
    //       headingsRenderedOnThisPage.push(headingID);
    //     }
    //     return '<h'+level+' class="markdown-heading" id="'+headingID+'">'+(textWithLineBreaks ? textWithLineBreaks : text)+'<a href="#'+headingID+'" class="markdown-link"></a></h'+level+'>\n';
    //   };
    // } else  {
    //   customRenderer.heading = function (text, level) {
    //     return '<h'+level+'>'+text+'</h'+level+'>';
    //   };
    // }

    // // Creating a custom codeblock renderer function to render mermaid code blocks (```mermaid```) without the added <pre> tags.
    // customRenderer.code = function(code) {
    //   if(code.match(/\<!-- __LANG=\%mermaid\%__ --\>/g)) {
    //     return '<code>'+_.escape(code)+'\n</code>';
    //   } else {
    //     return '<pre><code>'+_.escape(code)+'\n</code></pre>';
    //   }
    // };

    // // Creating a custom blockquote renderer function to render blockquotes as tip blockquotes.
    // customRenderer.blockquote = function(blockquoteHtml) {
    //   return `<blockquote purpose="tip"><img src="/images/icon-info-16x16@2x.png" alt="An icon indicating that this section has important information"><div class="d-block">\n${blockquoteHtml}\n</div></blockquote>`;
    // };

    // // Custom renderer function to enable checkboxes in Markdown lists.
    // customRenderer.listitem = function(innerHtml, hasCheckbox, isChecked) {
    //   if(!hasCheckbox){ // « If a list item does not have a checkbox, we'll render it normally.
    //     return `<li>${innerHtml}</li>`;
    //   } else if(isChecked) {// If this checkbox was checked in Markdown (- [x]), we'll add a disabled checked checkbox, and hide the original checkbox with CSS
    //     return `<li purpose="checklist-item"><input disabled type="checkbox" checked><span purpose="task">${innerHtml}</span></li>`;
    //   } else {// If the checkbox was not checked, we'll add a non-checked disabled checkbox, and hide the original checkbox with CSS.
    //     return `<li purpose="checklist-item"><input disabled type="checkbox"><span purpose="task">${innerHtml}</span></li>`;
    //   }
    // };

    // // Creating a custom table renderer to add Bootstrap's responsive table styles to markdown tables.
    // customRenderer.table = function(headerHtml, bodyHtml) {
    //   return `<div class="table-responsive-xl"><table class="table">\n<thead>\n${headerHtml}\n</thead>\n<tbody>${bodyHtml}\n</tbody>\n</table>\n</div>`;
    // };


    customRenderer.code = function(codeHTML) {
      return `<pre style=""><code>'+_.escape(code)+'</code></pre>`
    }
    customRenderer.blockquote = function(quoteHTML) {
      return `<blockquote>\n${quoteHTML}\n</blockquote>\n`;
    }
    customRenderer.heading = function(textHTML, level) {
      let inlineStyles;
      if(level === 1) {
        inlineStyles = 'font-weight: 800; font-size: 24px; line-height: 32px; margin-bottom: 16px;';
      } else if (level === 2) {
        inlineStyles = 'font-weight: 700; font-size: 20px; line-height: 28px; margin-bottom: 16px; margin-top: 32px;';
      } else if (level === 3) {
        inlineStyles = 'font-weight: 700; font-size: 20px; line-height: 24px; margin-bottom: 16px;';
      } else if (level === 4) {
        inlineStyles = 'font-weight: 700; font-size: 16px; line-height: 20px; margin-bottom: 16px;';
      }
      return `<h${level} style="${inlineStyles}">\n${textHTML}\n</h${level}>\n`;
    }
    customRenderer.hr = function() {
      return `<hr style="border-top: 2px solid #E2E4EA; margin-top: 16px; margin-bottom: 16px; width: 100%;">`;
    }
    customRenderer.list = function(bodyHTML, ordered, start) {
      if(ordered){
        return `<ol style="padding-left: 16px; margin-bottom: 32px;">\n${bodyHTML}</ol>\n`
      } else {
        return `<ul style="padding-left: 16px; margin-bottom: 32px;">\n${bodyHTML}</ul>\n`
      }
    }
    customRenderer.listitem = function(textHTMl, task, checked) {
      return `<li class="block-li" style="margin-bottom: 16px;">\n${textHTMl}\n</li>\n`
    }
    customRenderer.paragraph = function(text) {
      return `<p class="block-p" style="font-size: 16px; line-height: 24px; font-weight: 400; margin-bottom: 16px;">\n${text}\n</p>\n`;
    }
    customRenderer.table = function(headerHTML, bodyHTML) {
      return;
    }
    customRenderer.strong = function(textHTML) {
      return `<strong style="display: inline; font-weight: 700; font-size: 16px; line-height: 24px;">${textHTML}</strong>`;
    }
    customRenderer.em = function(textHTML) {
      return `<span style="display: inline; font-style: italic; font-size:16px;>${textHTML}</span>`
    }
    customRenderer.codespan = function(codeHTML) {
      return `<code style="display: inline; background: #F1F0FF; color: #192147; padding: 4px 8px; font-size: 13px; line-height: 16px; font-family: Source Code Pro;">${_.escape(codeHTML)}</code>`
    }
    customRenderer.link = function(href, title, textHTML) {
      return `<a style="display: inline; color: #6A67FE; font-size: 16px; text-decoration: none;" href="${href}">${textHTML}</a>`
    }
    customRenderer.image = function(href, title, textHTML) {
      let cannonicalLinkToImage = href.replace(/^(\.\.\/website\/assets)/gi, 'https://fleetdm.com')
      return `<img style="max-width: 100%; margin-top: 40px; margin-bottom: 40px;" src="${cannonicalLinkToImage}" alt="\n${title}\n">`
    }
    customRenderer.text = function(textHTML) {
      if(textHTML) {
        return `${textHTML}`;
      } else {
        return;
      }
    }
    markedOpts.renderer = customRenderer;
    // Now actually compile the markdown to HTML.
    marked(inputs.mdString, markedOpts, function afterwards (err, htmlString) {
      if (err) { return exits.error(err); }

      // If we're not allowing HTML, compile the input again with the `sanitize` option on.
      if (inputs.allowHtml === false) {
        markedOpts.sanitize = true;
        marked(inputs.mdString, markedOpts, function sanitized (err, sanitizedHtmlString) {
          if (err) { return exits.error(err); }

          // Now compare the unsanitized and the sanitized output, and if they're not the same,
          // leave through the `unsafeMarkdown` exit since it means that HTML tags were detected.
          if (htmlString !== sanitizedHtmlString) {
            return exits.unsafeMarkdown();
          }
          return exits.success(htmlString);
        });
      }
      else {
        return exits.success(htmlString);
      }
    });
  }


};
