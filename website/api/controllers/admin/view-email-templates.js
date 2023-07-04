module.exports = {


  friendlyName: 'View email templates',


  description: 'Display "Email templates" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/email-templates'
    }

  },


  fn: async function () {


    var path = require('path');

    // Sniff for top level email templates
    let templatePaths = await sails.helpers.fs.ls.with({
      dir: path.join(sails.config.paths.views, 'emails/'),
      depth: 1,
      includeDirs: false,
      includeSymlinks: false,
    });

    let markdownEmailPaths = await sails.helpers.fs.ls.with({
      dir: path.join(sails.config.paths.views, 'emails/newsletter'),
      depth: 99,
      includeDirs: false,
      includeSymlinks: false,
    });

    markdownEmailPaths = markdownEmailPaths.map((templatePath)=>{
      let relativePath = path.relative(path.join(sails.config.paths.views, 'emails/'), templatePath);
      let extension = path.extname(relativePath);
      return relativePath.split(extension)[0];
    });

    templatePaths = templatePaths.map((templatePath)=>{
      let relativePath = path.relative(path.join(sails.config.paths.views, 'emails/'), templatePath);
      let extension = path.extname(relativePath);
      return relativePath.split(extension)[0];
    });

    // Respond with view.
    return {
      templatePaths,
      markdownEmailPaths
    };

  }


};
