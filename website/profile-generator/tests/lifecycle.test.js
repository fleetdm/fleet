/**
 * Load the Sails app once for the whole mocha run.
 *
 * These are root hooks -- declared outside any `describe` -- so they run before the first test and
 * after the last one no matter which order mocha happens to load the files in.
 *
 * `sails.load` rather than `sails.lift`: nothing under test makes an HTTP request, and binding the
 * dev port would collide with a `sails lift` already running on this machine.  Nothing about the
 * models is overridden either, so the bootstrap behaves exactly as it does for `sails console` or
 * `sails run some-script` -- notably, it does not wipe the local development database.
 */
const sails = require('sails');
// Same config layer app.js and the `sails` CLI use: `.sailsrc`, and `sails_custom__anthropicSecret=…`
// style environment variables.  Without it, `sails.load` sees none of them, and a secret passed the
// way every other entry point takes it would be silently ignored.
const rc = require('sails/accessible/rc');


before(function(done) {
  this.timeout(60000);

  // Written out rather than deep-merged: `_` is a Sails global and does not exist yet.
  let configOverrides = rc('sails');
  // Compiling assets has nothing to do with these tests and is the slowest part of a lift.
  configOverrides.hooks = Object.assign({}, configOverrides.hooks, { grunt: false });
  // The generator helper and the prompt helper both report what they had to work around through
  // sails.log.warn, which this still lets through; the per-request info logging only gets in the way
  // of reading mocha's output.  Still overridable with `sails_log__level=info`, since the usual
  // reason to want it back is a lift that failed.
  configOverrides.log = Object.assign({ level: 'warn' }, configOverrides.log);

  sails.load(configOverrides, function(err) {
    if(err) { return done(err); }
    return done();
  });
});


after(function(done) {
  this.timeout(30000);
  if(!sails || !sails.lower) { return done(); }
  return sails.lower(done);
});
