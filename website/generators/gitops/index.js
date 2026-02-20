module.exports = {

  targets: {

    './it-and-security/README.md': { copy: './README.md.template' },
    './it-and-security/.github': { folder: {} },
    './it-and-security/.github/fleet-gitops': { folder: {} },
    './it-and-security/.github/fleet-gitops/action.yml': { copy: './github/fleet-gitops/action.yml.template' },
    './it-and-security/.github/fleet-gitops/gitops.sh': { copy: './github/fleet-gitops/gitops.sh.template' },
    './it-and-security/.github/workflows': { folder: {} },
    './it-and-security/.github/workflows/fleet-gitops-workflow.yml': { copy: './github/workflows/fleet-gitops-workflow.yml.template' },
    './it-and-security/.gitlab-ci.yml': { copy: './gitlab-ci.yml.template' },
    './it-and-security/.gitignore': { copy: './gitignore.template' },
    './it-and-security/default.yml': { copy: './default.yml.template' },
    './it-and-security/fleets/': { folder: {} },
    './it-and-security/fleets/workstations.yml': { copy: './fleets/workstations.yml.template' },
    './it-and-security/fleets/personal-mobile-devices.yml': { copy: './fleets/personal-mobile-devices.yml.template' },
    './it-and-security/fleets/company-owned-mobile-devices.yml': { copy: './fleets/company-owned-mobile-devices.yml.template' },
    './it-and-security/labels/': { folder: {} },
    './it-and-security/labels/apple-silicon-macos-hosts': { copy: './labels/apple-silicon-macos-hosts.yml.template' },
    './it-and-security/labels/x86-based-windows-hosts': { copy: './labels/x86-based-windows-hosts.yml.template' },
    './it-and-security/labels/arm-based-windows-hosts': { copy: './labels/arm-based-windows-hosts.yml.template' },
    './it-and-security/labels/debian-based-linux-hosts': { copy: './labels/debian-based-linux-hosts.yml.template' },
    './it-and-security/icons/': { folder: {} },
    './it-and-security/icons/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/': { folder: {} },
    './it-and-security/platforms/linux': { folder: {} },
    './it-and-security/platforms/linux/policies/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/linux/reports/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/linux/scripts/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/linux/software/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/windows': { folder: {} },
    './it-and-security/platforms/windows/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/windows/policies/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/windows/reports/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/windows/scripts/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/windows/software/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos': { folder: {} },
    './it-and-security/platforms/macos/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/declaration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/enrollment-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/commands/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/policies/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/reports/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/scripts/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/macos/software/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/ios': { folder: {} },
    './it-and-security/platforms/ios/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/ios/declaration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/ipados': { folder: {} },
    './it-and-security/platforms/ipados/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/ipados/declaration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/tvos': { folder: {} },
    './it-and-security/platforms/tvos/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/android': { folder: {} },
    './it-and-security/platforms/android/configuration-profiles/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/android/managed-app-configurations/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/all/': { folder: {} },
    './it-and-security/platforms/all/reports/.gitkeep': { copy: './gitkeep.template' },
    './it-and-security/platforms/all/policies/.gitkeep': { copy: './gitkeep.template' },

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
