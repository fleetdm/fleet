module.exports = {

  targets: {

    './it-and-security/.gitignore': { copy: './default.yml.template' },
    './it-and-security/README.md': { copy: './default.yml.template' },
    './it-and-security/default.yml': { copy: './default.yml.template' },
    './it-and-security/fleets/': { folder: {} },
    './it-and-security/fleets/workstations.yml': { copy: './default.yml.template' },
    './it-and-security/fleets/personal-mobile-devices.yml': { copy: './default.yml.template' },
    './it-and-security/fleets/company-owned-mobile-devices.yml': { copy: './default.yml.template' },
    './it-and-security/labels/': { folder: {} },
    './it-and-security/icons/': { folder: {} },
    './it-and-security/platforms/': { folder: {} },
    './it-and-security/platforms/linux': { folder: {} },
    './it-and-security/platforms/windows': { folder: {} },
    './it-and-security/platforms/macos': { folder: {} },
    './it-and-security/platforms/ios': { folder: {} },
    './it-and-security/platforms/ipados': { folder: {} },
    './it-and-security/platforms/tvos': { folder: {} },
    './it-and-security/platforms/android': { folder: {} },

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // • e.g. create a folder:
    // ```
    // './hey_look_a_folder': { folder: {} }
    // ```
    //
    // • e.g. create a dynamically-named file relative to `scope.rootPath`
    // (defined by the `filename` scope variable).
    //
    // The `template` helper reads the specified template, making the
    // entire scope available to it (uses underscore/JST/ejs syntax).
    // Then the file is copied into the specified destination (on the left).
    // ```
    // './:filename': { template: 'example.template.js' },
    // ```
    //
    // • See https://sailsjs.com/docs/concepts/extending-sails/generators for more documentation.
    // (Or visit https://sailsjs.com/support and talk to a maintainer of a core or community generator.)
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

  },


  /**
   * The absolute path to the `templates` for this generator
   * (for use with the `template` and `copy` builtins)
   *
   * @type {String}
   */
  templatesDirectory: require('path').resolve(__dirname, './templates')

};
