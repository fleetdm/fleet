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
      defaultsTo: true
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

    if (inputs.addIdsToHeadings === true) {
      var headingsRenderedOnThisPage = [];
      customRenderer.heading = function (text, level) {
        // If the heading has underscores and no spaces (e.g. osquery_async_host_collect_log_stats_interval) we'll add optional linebreaks before each underscore
        var textWithLineBreaks;
        if(text.match(/\S(\w+\_\S)+(\w\S)+/g) && !text.match(/\s/g)){
          textWithLineBreaks = text.replace(/(\_)/g, '&#8203;_');
        }
        var headingID = _.kebabCase(_.unescape(text.toLowerCase()).replace(/[\’\']/g, ''));
        if(!_.contains(headingsRenderedOnThisPage, headingID)){
          headingsRenderedOnThisPage.push(headingID);
        } else {
          headingID = sails.helpers.strings.ensureUniq(headingID, headingsRenderedOnThisPage);
          headingsRenderedOnThisPage.push(headingID);
        }
        return '<h'+level+' class="markdown-heading" id="'+headingID+'">'+(textWithLineBreaks ? textWithLineBreaks : text)+'<a href="#'+headingID+'" class="markdown-link"></a></h'+level+'>\n';
      };
    } else  {
      customRenderer.heading = function (text, level) {
        var textWithLineBreaks;
        if(text.match(/\S(\w+\_\S)+(\w\S)+/g) && !text.match(/\s/g)){
          textWithLineBreaks = text.replace(/(\_)/g, '&#8203;_');
        }
        return '<h'+level+'>'+(textWithLineBreaks ? textWithLineBreaks : text)+'</h'+level+'>';
      };
    }

    // Creating a custom codeblock renderer function to add syntax highlighting keywords and render mermaid code blocks (```mermaid```) without the added <pre> tags.
    customRenderer.code = function(code, infostring) {
      if(infostring === 'mermaid') {
        return `<code class="mermaid">${_.escape(code)}</code>`;
      } else if(infostring === 'js') {// Interpret `js` as `javascript`
        return `<pre><code class="hljs javascript" v-pre>${_.escape(code)}</code></pre>`;
      } else if(infostring === 'bash' || infostring === 'sh') {// Interpret `sh` and `bash` as `bash`
        return `<pre><code class="hljs bash" v-pre>${_.escape(code)}</code></pre>`;
      } else if(infostring !== '') {// leaving the code language as-is if the infoString is anything else.
        return `<pre><code class="hljs ${_.escape(infostring)}" v-pre>${_.escape(code)}</code></pre>`;
      } else {// When unspecified, default to `text`
        return `<pre><code class="nohighlight" v-pre>${_.escape(code)}</code></pre>`;
      }
    };

    // Creating a custom blockquote renderer function to render blockquotes as tip blockquotes.
    customRenderer.blockquote = function(blockquoteHtml) {
      return `<blockquote purpose="tip"><img src="/images/icon-info-16x16@2x.png" alt="An icon indicating that this section has important information"><div class="d-block">\n${blockquoteHtml}\n</div></blockquote>`;
    };

    // Custom renderer function to enable checkboxes in Markdown lists.
    customRenderer.listitem = function(innerHtml, hasCheckbox, isChecked) {
      if(!hasCheckbox){ // « If a list item does not have a checkbox, we'll render it normally.
        return `<li>${innerHtml}</li>`;
      } else if(isChecked) {// If this checkbox was checked in Markdown (- [x]), we'll add a disabled checked checkbox, and hide the original checkbox with CSS
        return `<li purpose="checklist-item"><input disabled type="checkbox" checked><span purpose="task">${innerHtml}</span></li>`;
      } else {// If the checkbox was not checked, we'll add a non-checked disabled checkbox, and hide the original checkbox with CSS.
        return `<li purpose="checklist-item"><input disabled type="checkbox"><span purpose="task">${innerHtml}</span></li>`;
      }
    };

    // Creating a custom table renderer to add Bootstrap's responsive table styles to markdown tables.
    customRenderer.table = function(headerHtml, bodyHtml) {
      return `<div class="table-responsive"><table class="table">\n<thead>\n${headerHtml}\n</thead>\n<tbody>${bodyHtml}\n</tbody>\n</table>\n</div>`;
    };

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
