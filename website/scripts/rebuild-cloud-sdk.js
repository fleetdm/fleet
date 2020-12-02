module.exports = {

  friendlyName: 'Rebuild Cloud SDK',


  description: 'Regenerate the configuration for the "Cloud SDK" -- the JavaScript module used for AJAX and WebSockets.',


  fn: async function(){

    var path = require('path');

    var endpointsByMethodName = {};
    var extraEndpointsOnlyForTestsByMethodName = {};

    for (let address in sails.config.routes) {
      let target = sails.config.routes[address];

      // If the route target is an array, then only consider
      // the very last sub-target in the array.
      if (_.isArray(target)) {
        target = _.last(target);
      }//ﬁ

      // Skip redirects
      // (Note that, by doing this, we also skip traditional shorthand
      // -- that's ok though.)
      if (_.isString(target)) {
        continue;
      }

      // Skip routes whose target doesn't contain `action` for any
      // other miscellaneous reason.
      if (!target.action) {
        continue;
      }

      // Just about everything else gets a Cloud SDK method.

      // We determine its name using the bare action name.
      var bareActionName = _.last(target.action.split(/\//));
      var methodName = _.camelCase(bareActionName);
      var expandedAddress = sails.getRouteFor(target);

      // Skip routes that just serve views.
      // (but still generate them for use in tests, for convenience)
      if (target.view || (bareActionName.match(/^view-/))) {
        extraEndpointsOnlyForTestsByMethodName[methodName] = {
          verb: (expandedAddress.method||'get').toUpperCase(),
          url: expandedAddress.url
        };
        continue;
      }//•

      endpointsByMethodName[methodName] = {
        verb: (expandedAddress.method||'get').toUpperCase(),
        url: expandedAddress.url,
      };

      // If this is an actions2 action, then determine appropriate serial usage.
      // (deduced the same way as helpers)
      // > If there is no such action for some reason, then don't compile a
      // > method for this one.
      var requestable = sails.getActions()[target.action];
      if (!requestable) {
        sails.log.warn('Skipping unrecognized action: `'+target.action+'`');
        continue;
      }
      var def = requestable.toJSON && requestable.toJSON();
      if (def && def.fn) {
        if (def.args !== undefined) {
          endpointsByMethodName[methodName].args = def.args;
        } else {
          endpointsByMethodName[methodName].args = _.reduce(def.inputs, (args, inputDef, inputCodeName)=>{
            args.push(inputCodeName);
            return args;
          }, []);
        }
      }

      // And we determine whether it needs to communicate over WebSockets
      // by checking for an additional property in the route target.
      if (target.isSocket) {
        endpointsByMethodName[methodName].protocol = 'io.socket';
      }
    }//∞

    // Smash and rewrite the `cloud.setup.js` file in the assets folder to
    // reflect the latest set of available cloud actions exposed by this Sails
    // app (as determined by its routes above)
    await sails.helpers.fs.write.with({
      destination: path.resolve(sails.config.appPath, 'assets/js/cloud.setup.js'),
      force: true,
      string: ``+
`/**
 * cloud.setup.js
 *
 * Configuration for this Sails app's generated browser SDK ("Cloud").
 *
 * Above all, the purpose of this file is to provide endpoint definitions,
 * each of which corresponds with one particular route+action on the server.
 *
 * > This file was automatically generated.
 * > (To regenerate, run \`sails run rebuild-cloud-sdk\`)
 */

Cloud.setup({

  /* eslint-disable */
  methods: ${JSON.stringify(endpointsByMethodName)}
  /* eslint-enable */

});`+
      `\n`
    });

    // Also, if a `test/` folder exists, set up a barebones bounce of this data
    // as a JSON file inside of it, for testing purposes:
    var hasTestFolder = await sails.helpers.fs.exists(path.resolve(sails.config.appPath, 'test/'));
    if (hasTestFolder) {
      await sails.helpers.fs.write.with({
        destination: path.resolve(sails.config.appPath, 'test/private/CLOUD_SDK_METHODS.json'),
        string: JSON.stringify(_.extend(endpointsByMethodName, extraEndpointsOnlyForTestsByMethodName)),
        force: true
      });
    }

    sails.log.info('--');
    sails.log.info('Successfully rebuilt Cloud SDK for use in the browser.');
    sails.log.info('(and CLOUD_SDK_METHODS.json for use in automated tests)');

  }

};
