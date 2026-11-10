/**
 * Registering a Fleet server to use the configuration profile generator, checked through the registered
 * route as a Sails virtual request.  Nothing here calls a model.
 */
const assert = require('assert');
const crypto = require('crypto');


describe('registering a Fleet server for the profile generator', function() {

  const ROUTE = 'POST /api/profile-generator/v1/register';
  const CLIENT_IP = '203.0.113.7';
  let createdRegistrationIds = [];
  let originalRoute;

  // Virtual requests carry no client IP address, and sails.request() passes on only the method, URL, body
  // and headers.  The router copies any other property of the request it is given onto the request it
  // builds, so give these requests one there.
  before(function() {
    originalRoute = sails.router.route;
    sails.router.route = function(req, res) {
      if(req && _.startsWith(req.url, '/api/profile-generator/v1/register')) {
        req.ip = CLIENT_IP;
      }
      return originalRoute.call(this, req, res);
    };
  });

  // A request with no marketingAttribution cookie makes the custom hook set one, which a virtual response
  // cannot do, so every request carries one.  No CSRF token or session is sent: Fleet servers have neither.
  let register = (data)=>{
    return new Promise((resolve)=>{
      sails.request({ method: 'POST', url: '/api/profile-generator/v1/register', data, headers: { cookie: 'marketingAttribution=%7B%7D' } }, (err, res, body)=>{
        resolve(err ? { status: err.status, body: err.body } : { status: res.statusCode, body });
      });
    });
  };
  let hashOf = (secret)=>{ return crypto.createHash('sha256').update(secret).digest('hex'); };
  let registrationsFor = async (fleetServerUrl)=>{
    let registrations = await ProfileGeneratorRegistration.find({ fleetServerUrl });
    createdRegistrationIds = _.union(createdRegistrationIds, _.pluck(registrations, 'id'));
    return registrations;
  };
  // A URL no other test or local data uses, so each test sees only its own rows.
  let uniqueFleetServerUrl = ()=>{ return `https://fleet-${crypto.randomBytes(6).toString('hex')}.example.com`; };

  after(async function() {
    sails.router.route = originalRoute;
    if(createdRegistrationIds.length > 0) {
      await ProfileGeneratorRegistration.destroy({ id: { in: createdRegistrationIds } });
    }
  });

  it('is public and needs no CSRF token', function() {
    assert.strictEqual(sails.config.routes[ROUTE].csrf, false);
    // Sails normalizes a policy of `true` to `[true]` when it loads.
    assert.deepStrictEqual(sails.config.policies['profile-generator/register-one-fleet-instance'], [true]);
  });

  it('returns a secret and stores the URL, the client IP, a hash of the secret, and no usage', async function() {
    let fleetServerUrl = uniqueFleetServerUrl();
    let response = await register({ fleetServerUrl });
    assert.strictEqual(response.status, 200);
    assert.deepStrictEqual(_.keys(response.body), ['fleetServerSecret']);
    let secret = response.body.fleetServerSecret;
    assert.match(secret, /^[0-9a-f]{64}$/);

    let registrations = await registrationsFor(fleetServerUrl);
    assert.strictEqual(registrations.length, 1);
    let registration = registrations[0];
    assert.strictEqual(registration.fleetServerSecretHash, hashOf(secret));
    assert.strictEqual(registration.registeredFromIpAddress, CLIENT_IP);
    assert.strictEqual(registration.usageMonth, '');
    assert.strictEqual(registration.profileGenerationsThisMonth, 0);
    assert.strictEqual(registration.nameAndDescriptionRequestsThisMonth, 0);
    // The plaintext secret is not stored in any column.
    assert.ok(!JSON.stringify(registration).includes(secret));
  });

  it('can find a registration from the secret alone', async function() {
    let fleetServerUrl = uniqueFleetServerUrl();
    let response = await register({ fleetServerUrl });
    await registrationsFor(fleetServerUrl);
    let registration = await ProfileGeneratorRegistration.findOne({ fleetServerSecretHash: hashOf(response.body.fleetServerSecret) });
    assert.strictEqual(registration.fleetServerUrl, fleetServerUrl);
  });

  it('creates a new registration with a new secret each time the same URL registers', async function() {
    let fleetServerUrl = uniqueFleetServerUrl();
    let first = await register({ fleetServerUrl });
    let second = await register({ fleetServerUrl });
    assert.strictEqual(first.status, 200);
    assert.strictEqual(second.status, 200);
    assert.notStrictEqual(first.body.fleetServerSecret, second.body.fleetServerSecret);

    let registrations = await registrationsFor(fleetServerUrl);
    assert.strictEqual(registrations.length, 2);
    assert.deepStrictEqual(
      _.pluck(registrations, 'fleetServerSecretHash').sort(),
      [hashOf(first.body.fleetServerSecret), hashOf(second.body.fleetServerSecret)].sort()
    );
  });

  it('accepts an http:// URL, as Fleet does for its own server URL', async function() {
    for (let fleetServerUrl of [`http://fleet-${crypto.randomBytes(6).toString('hex')}.example.com`, 'http://localhost:8080']) {
      let response = await register({ fleetServerUrl });
      assert.strictEqual(response.status, 200, `expected a 200 for ${fleetServerUrl}`);
      let registration = await ProfileGeneratorRegistration.findOne({ fleetServerSecretHash: hashOf(response.body.fleetServerSecret) });
      createdRegistrationIds.push(registration.id);
      assert.strictEqual(registration.fleetServerUrl, fleetServerUrl);
    }
  });

  it('is a 400 and stores nothing for a URL that is not http:// or https:// with a host', async function() {
    let countBefore = await ProfileGeneratorRegistration.count();
    for (let fleetServerUrl of ['not a url', 'fleet.example.com', 'ftp://fleet.example.com', 'foo:bar', 'http://', 'https://', '']) {
      let response = await register({ fleetServerUrl });
      assert.strictEqual(response.status, 400, `expected a 400 for ${JSON.stringify(fleetServerUrl)}`);
    }
    let missing = await register({});
    assert.strictEqual(missing.status, 400);
    assert.strictEqual(await ProfileGeneratorRegistration.count(), countBefore);
  });

});
