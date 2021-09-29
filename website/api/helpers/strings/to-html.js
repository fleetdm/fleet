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
    var marked = require('marked');

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

    if (inputs.addIdsToHeadings === false) {
      var renderer = new marked.Renderer();
      renderer.heading = function (text, level) {
        return '<h'+level+'>'+text+'</h'+level+'>';
      };
      markedOpts.renderer = renderer;
    }

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
