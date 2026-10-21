/**
 * The .mobileconfig, Windows CSP, and WLAN profile builders, checked without calling a model: each is given the
 * settings a model would return and the document it writes is checked against Apple's and Microsoft's rules.
 */
const assert = require('assert');


/**
 * Whether a document's tags nest and close properly.  CDATA sections, processing instructions, and the DOCTYPE are
 * skipped, and a bare & outside them must start an entity.  The builders write no comments, so one counts as stray
 * markup.
 *
 * @param  {String} xml
 * @returns {String|undefined}  What is wrong, or undefined for a well-formed document.
 */
function whyNotWellFormed(xml) {
  let text = xml.replace(/<!\[CDATA\[[\s\S]*?\]\]>/g, '').replace(/<\?[\s\S]*?\?>/g, '').replace(/<!DOCTYPE[^>]*>/g, '');
  let openTags = [];
  let tagPattern = /<(\/?)([A-Za-z_][\w.:-]*)((?:\s+[\w:.-]+\s*=\s*(?:"[^"<]*"|'[^'<]*'))*)\s*(\/?)>/g;
  let lastIndex = 0;
  let match;
  while ((match = tagPattern.exec(text))) {
    let between = text.slice(lastIndex, match.index);
    if(/</.test(between) || /&(?!(?:amp|lt|gt|quot|apos|#\d+|#x[0-9a-fA-F]+);)/.test(between)) {
      return `stray markup before <${match[1]}${match[2]}>: ${JSON.stringify(between.slice(0, 60))}`;
    }
    lastIndex = tagPattern.lastIndex;
    let [, isClosing, name, , isSelfClosing] = match;
    if(isClosing) {
      if(openTags.pop() !== name) {
        return `</${name}> closes something else`;
      }
    } else if(!isSelfClosing) {
      openTags.push(name);
    }
  }
  if(/[<&]/.test(text.slice(lastIndex).replace(/&(?:amp|lt|gt|quot|apos);/g, ''))) {
    return 'stray markup after the last tag';
  }
  return openTags.length > 0 ? `<${openTags.pop()}> is never closed` : undefined;
}


/**
 * @param  {String} text
 * @param  {Array} substrings
 * @returns {Boolean}  Whether every substring appears, in the order given.
 */
function appearsInOrder(text, substrings) {
  let from = 0;
  return _.every(substrings, (substring)=>{
    let at = text.indexOf(substring, from);
    from = at + substring.length;
    return at !== -1;
  });
}


describe('profile builders', function() {

  describe('.mobileconfig from payload settings', function() {

    let build = (payloads, extra)=>{
      return sails.helpers.buildMobileconfigFromPayloadSettings.with(Object.assign({
        payloads,
        profileIdentifier: 'com.example.test',
        profileDisplayName: 'Test',
      }, extra));
    };

    it('types each value from Apple\'s schema rather than from JSON', function() {
      let built = build([
        { payloadType: 'com.apple.mobiledevice.passwordpolicy', settings: { forcePIN: 1, minLength: '8', allowSimple: 'FALSE' } },
        { payloadType: '.GlobalPreferences', settings: { 'com.apple.autologout.AutoLogOutDelay': 600 } },
        { payloadType: 'com.apple.system-extension-policy', settings: { AllowedSystemExtensions: { ABCDE12345: ['com.acme.agent.extension'] } } },
      ], { profileSettings: { RemovalDate: '2027-01-01T08:30:00.000Z' } });
      assert.deepStrictEqual(built.unusableValues, []);
      assert.strictEqual(whyNotWellFormed(built.profile), undefined);
      let profile = built.profile.replace(/\s+/g, '');
      assert.ok(_.contains(profile, '<key>forcePIN</key><true/>'));
      assert.ok(_.contains(profile, '<key>minLength</key><integer>8</integer>'));
      assert.ok(_.contains(profile, '<key>allowSimple</key><false/>'));
      assert.ok(_.contains(profile, '<key>com.apple.autologout.AutoLogOutDelay</key><real>600</real>'));
      assert.ok(_.contains(profile, '<key>ABCDE12345</key><array><string>com.acme.agent.extension</string></array>'));
      assert.ok(_.contains(profile, '<key>RemovalDate</key><date>2027-01-01T08:30:00Z</date>'));
    });

    it('writes data as base64, stripping PEM armor and encoding plain text', function() {
      let built = build([
        { payloadType: 'com.apple.security.root', settings: { PayloadCertificateFileName: 'root.cer', PayloadContent: '-----BEGIN CERTIFICATE-----\nTUlJQmlqQ0NB\n-----END CERTIFICATE-----' } },
        { payloadType: 'com.apple.vpn.managed', settings: { VPNType: 'IPSec', IPSec: { SharedSecret: 'S3cret&<x>' } } },
      ]);
      assert.deepStrictEqual(built.unusableValues, []);
      assert.strictEqual(whyNotWellFormed(built.profile), undefined);
      assert.ok(_.contains(built.profile, '<data>TUlJQmlqQ0NB</data>'));
      assert.ok(_.contains(built.profile, `<data>${Buffer.from('S3cret&<x>').toString('base64')}</data>`));
    });

    it('writes root-level keys on the root dict, including one found inside a payload', function() {
      let built = build([
        { payloadType: 'com.apple.applicationaccess', settings: { allowCamera: false, PayloadScope: 'System' } },
      ], { profileSettings: { PayloadRemovalDisallowed: true, PayloadUUID: 'NOT-SETTABLE' } });
      assert.deepStrictEqual(built.unusableValues, []);
      let rootPart = built.profile.slice(built.profile.lastIndexOf('</array>'));
      assert.ok(_.contains(rootPart.replace(/\s+/g, ''), '<key>PayloadRemovalDisallowed</key><true/>'));
      assert.ok(_.contains(rootPart.replace(/\s+/g, ''), '<key>PayloadScope</key><string>System</string>'));
      assert.ok(!_.contains(rootPart, 'NOT-SETTABLE'));
      assert.deepStrictEqual(_.pluck(built.settingsEnforced, 'name').sort(), ['PayloadRemovalDisallowed', 'PayloadScope', 'allowCamera']);
    });

    it('reports a value that does not fit its key\'s type instead of writing it', function() {
      let built = build([
        { payloadType: 'com.apple.applicationaccess', settings: { allowCamera: 'disabled', allowAirDrop: [false] } },
        { payloadType: 'com.apple.mobiledevice.passwordpolicy', settings: { minLength: 8.5 } },
      ], { profileSettings: { RemovalDate: 'next tuesday', PayloadScope: { level: 'System' } } });
      assert.deepStrictEqual(built.unusableValues.sort(), [
        'PayloadScope takes text, not {"level":"System"}.',
        'RemovalDate takes a date, not "next tuesday".',
        'com.apple.applicationaccess/allowAirDrop takes true or false, not [false].',
        'com.apple.applicationaccess/allowCamera takes true or false, not "disabled".',
        'com.apple.mobiledevice.passwordpolicy/minLength takes a whole number, not 8.5.',
      ]);
    });

    it('types keys the schema does not list the way JSON typed them', function() {
      let built = build([
        { payloadType: 'com.google.Chrome', settings: { RestoreOnStartup: 4, SomeRatio: 1.5, BrowserSignin: true, HomepageLocation: 'https://example.com' } },
      ]);
      assert.deepStrictEqual(built.unusableValues, []);
      let profile = built.profile.replace(/\s+/g, '');
      assert.ok(_.contains(profile, '<key>RestoreOnStartup</key><integer>4</integer>'));
      assert.ok(_.contains(profile, '<key>SomeRatio</key><real>1.5</real>'));
      assert.ok(_.contains(profile, '<key>BrowserSignin</key><true/>'));
    });

    it('leaves a null setting out of the profile and out of what it enforces', function() {
      let built = build([{ payloadType: 'com.apple.applicationaccess', settings: { allowCamera: false, allowAirDrop: null } }]);
      assert.deepStrictEqual(built.unusableValues, []);
      assert.ok(!_.contains(built.profile, 'allowAirDrop'));
      assert.deepStrictEqual(_.pluck(built.settingsEnforced, 'name'), ['allowCamera']);
    });

    it('requires each PayloadUUID to be a UUID that no other payload uses', function() {
      let uuid = 'AAAAAAAA-1111-2222-3333-444444444444';
      assert.deepStrictEqual(build([
        { payloadType: 'com.apple.security.root', payloadUUID: uuid, settings: {} },
        { payloadType: 'com.apple.security.pkcs1', settings: { PayloadUUID: uuid.toLowerCase() } },
      ]).unusableValues, [`Two payloads share the PayloadUUID ${uuid.toLowerCase()}.`]);
      assert.deepStrictEqual(build([
        { payloadType: 'com.apple.applicationaccess', payloadUUID: 12345, settings: { allowCamera: false } },
      ]).unusableValues, ['The com.apple.applicationaccess payload\'s PayloadUUID, 12345, is not a UUID.']);
    });

  });


  describe('Windows profile from commands', function() {

    let build = (commands, atomic)=>{
      return sails.helpers.buildWindowsProfileFromCommands.with({ commands, atomic: !!atomic });
    };
    // A Windows profile is a sequence of commands, so it is checked inside one element.
    let wellFormed = (profile)=>{ return whyNotWellFormed(`<profile>${profile}</profile>`); };

    it('takes each Format from the CSP reference, whatever the model gave', function() {
      let built = build([
        { verb: 'replace', locURI: './Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera', data: 0, format: 'bool' },
        { verb: 'Replace', locURI: './Vendor/MSFT/Firewall/MdmStore/PublicProfile/AllowLocalIpsecPolicyMerge', data: 0 },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/WiFi/Profile/Corp%20Net/Proxy', data: 'proxy.example.com:8080' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation', data: '<enabled/><data id="HomepageLocation" value="https://example.com"/>' },
      ]);
      assert.deepStrictEqual(built.unusableCommands, []);
      assert.strictEqual(wellFormed(built.profile), undefined);
      let profile = built.profile.replace(/\s+/g, '');
      assert.ok(appearsInOrder(profile, ['<Replace>', 'int</Format>', 'Camera/AllowCamera</LocURI>', '<Data>0</Data>']));
      assert.ok(appearsInOrder(profile, ['bool</Format>', 'AllowLocalIpsecPolicyMerge</LocURI>', '<Data>false</Data>']));
      assert.ok(appearsInOrder(profile, ['chr</Format>', 'Corp%20Net/Proxy</LocURI>', '<Data>proxy.example.com:8080</Data>']));
      assert.ok(_.contains(profile, '<Data><![CDATA[<enabled/><dataid="HomepageLocation"value="https://example.com"/>]]></Data>'));
      assert.ok(!/<Atomic>/.test(profile));
    });

    it('takes a format from the model only when it is a SyncML format, for a node the reference does not list', function() {
      let built = build([
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Example/Unlisted/Count', data: 5, format: 'integer' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Example/Unlisted/Enabled', data: true, format: 'Bool' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Example/Unlisted/Name', data: 'x', format: 'string' },
      ]);
      assert.deepStrictEqual(built.unusableCommands, []);
      let formats = Array.from(built.profile.matchAll(/<Format xmlns="syncml:metinf">([^<]*)<\/Format>/g), (match)=>{ return match[1]; });
      assert.deepStrictEqual(formats, ['int', 'bool', 'chr']);
    });

    it('rejects an Exec, which runs a one-time command, except to enroll a SCEP certificate', function() {
      let built = build([{ verb: 'Exec', locURI: './Device/Vendor/MSFT/Defender/Scan/Quick' }]);
      assert.deepStrictEqual(built.unusableCommands, ['./Device/Vendor/MSFT/Defender/Scan/Quick runs a one-time command, which a profile cannot do.']);
      let enroll = build([{ verb: 'Exec', locURI: './Device/Vendor/MSFT/ClientCertificateInstall/SCEP/Corp/Install/Enroll' }]);
      assert.deepStrictEqual(enroll.unusableCommands, []);
      assert.ok(/^<Exec>/.test(enroll.profile));
    });

    it('rejects data that does not fit its node, and a verb a profile cannot carry', function() {
      let built = build([
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera', data: true },
        { verb: 'Replace', locURI: './Vendor/MSFT/Firewall/MdmStore/PublicProfile/AllowLocalIpsecPolicyMerge', data: 'maybe' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/WiFi/Profile/Corp/Proxy' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/WiFi/Profile/Corp/Proxy', data: { host: 'proxy' } },
        { verb: 'Delete', locURI: './Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera' },
      ]);
      assert.strictEqual(built.unusableCommands.length, 5, built.unusableCommands.join('\n'));
      assert.ok(_.contains(built.unusableCommands[0], 'takes a whole number, not true'));
      assert.ok(_.contains(built.unusableCommands[1], 'takes true or false, not "maybe"'));
      assert.ok(_.contains(built.unusableCommands[2], 'needs a value'));
      assert.ok(_.contains(built.unusableCommands[3], 'not an object'));
      assert.ok(_.contains(built.unusableCommands[4], '"Delete" is not a command a profile can carry'));
    });

    it('creates an interior node with Format node, and wraps a node Microsoft requires in <Atomic>', function() {
      let built = build([
        { verb: 'Add', locURI: './Device/Vendor/MSFT/VPNv2/CorpVPN' },
        { verb: 'Add', locURI: './Device/Vendor/MSFT/VPNv2/CorpVPN/ProfileXML', data: '<VPNProfile><NativeProfile><Servers>vpn.example.com</Servers><RoutingPolicyType>SplitTunnel</RoutingPolicyType></NativeProfile></VPNProfile>' },
      ]);
      assert.deepStrictEqual(built.unusableCommands, []);
      assert.strictEqual(wellFormed(built.profile), undefined);
      let profile = built.profile.replace(/\s+/g, '');
      assert.ok(/^<Atomic>/.test(profile) && /<\/Atomic>$/.test(profile));
      assert.ok(appearsInOrder(profile, ['<Formatxmlns="syncml:metinf">node</Format>', 'VPNv2/CorpVPN</LocURI>', '</Item>']));
      assert.ok(!/VPNv2\/CorpVPN<\/LocURI><\/Target><Data>/.test(profile));
      assert.ok(_.contains(profile, '<Data><![CDATA[<VPNProfile><NativeProfile><Servers>vpn.example.com</Servers>'));
      assert.strictEqual(build([{ verb: 'Add', locURI: './Device/Vendor/MSFT/VPNv2/CorpVPN', data: 'x' }]).unusableCommands.length, 1);
    });

    it('wraps every command in one <Atomic> when asked to', function() {
      let built = build([
        { locURI: './Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera', data: 0 },
        { locURI: './Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana', data: 0 },
      ], true);
      let profile = built.profile.replace(/\s+/g, '');
      assert.strictEqual(profile.match(/<Atomic>/g).length, 1);
      assert.ok(/^<Atomic><Replace>.*<\/Replace><\/Atomic>$/.test(profile));
    });

    it('keeps a value well-formed whether or not it is wrapped in CDATA', function() {
      let built = build([
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/InteractiveLogon_MessageTextForUsersAttemptingToLogOn', data: 'Use ends here ]]> really' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/InteractiveLogon_MessageTitleForUsersAttemptingToLogOn', data: 'A & B ]]> C' },
        { verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/DeviceLock/PreventEnablingLockScreenCamera', data: '<![CDATA[<enabled/>]]>' },
      ]);
      assert.deepStrictEqual(built.unusableCommands, []);
      assert.strictEqual(wellFormed(built.profile), undefined);
      assert.ok(_.contains(built.profile, '<Data>Use ends here ]]&gt; really</Data>'));
      assert.ok(_.contains(built.profile, '<Data><![CDATA[A & B ]]]]><![CDATA[> C]]></Data>'));
      assert.ok(_.contains(built.profile, '<Data><![CDATA[<enabled/>]]></Data>'));
    });

    it('adds a Wi-Fi network with one Add of its WlanXml, noting a missing server to trust', function() {
      let built = build([
        { verb: 'Add', wifi: { ssid: 'A Network', security: 'WPA2Personal', passphrase: 'aaaaaaapassword' } },
        { verb: 'Add', wifi: { ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'PEAP' }, scope: 'User' } },
        { verb: 'Add', wifi: { ssid: 'Lab', security: 'WPA2Enterprise', eap: { method: 'EAP-TLS', serverNames: ['radius.example.com'] } } },
      ]);
      assert.deepStrictEqual(built.unusableCommands, []);
      assert.strictEqual(wellFormed(built.profile), undefined);
      assert.ok(_.contains(built.profile, '<LocURI>./Device/Vendor/MSFT/WiFi/Profile/A%20Network/WlanXml</LocURI>'));
      assert.ok(_.contains(built.profile, '<LocURI>./User/Vendor/MSFT/WiFi/Profile/Corp/WlanXml</LocURI>'));
      assert.strictEqual(built.profile.match(/<Add>/g).length, 3);
      assert.deepStrictEqual(built.deliveryNotes, ['Add the RADIUS server names and root CA thumbprint for Corp before deploying it, or users will be asked to trust the server\'s certificate when they first connect.']);
    });

    it('reports a Wi-Fi network it cannot write, with the reason', function() {
      let built = build([
        { verb: 'Add', wifi: { ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'EAP-FAST' } } },
        { verb: 'Add', wifi: { ssid: 'Corp', security: 'WPA2' } },
      ]);
      assert.strictEqual(built.unusableCommands.length, 2);
      assert.ok(_.contains(built.unusableCommands[0], 'needs an EAP method of PEAP, TLS, TTLS, not FAST'));
      assert.ok(_.contains(built.unusableCommands[1], 'The Wi-Fi network "Corp" could not be written'));
    });

  });


  describe('WLAN profile XML', function() {

    let wlan = (inputs)=>{
      return sails.helpers.buildWlanProfileXml.with(inputs);
    };
    let element = (xml, name)=>{
      let match = xml.match(new RegExp(`<${name}(?:\\s[^>]*)?>([^<]*)</${name}>`));
      return match ? match[1] : undefined;
    };

    it('writes a personal network in the order WLAN_profile requires', function() {
      let xml = wlan({ ssid: 'Café & <Guest>', security: 'WPA2Personal', passphrase: 'p&ss<word>', hidden: true, connectAutomatically: false });
      assert.strictEqual(whyNotWellFormed(xml), undefined);
      assert.ok(appearsInOrder(xml, ['<name>', '<SSIDConfig><SSID><hex>', '</hex><name>', '</SSID><nonBroadcast>true</nonBroadcast>', '<connectionType>ESS', '<connectionMode>manual', '<MSM><security><authEncryption>', '</authEncryption><sharedKey>']));
      assert.strictEqual(element(xml, 'hex'), Buffer.from('Café & <Guest>', 'utf8').toString('hex').toUpperCase());
      assert.ok(_.contains(xml, '<name>Café &amp; &lt;Guest&gt;</name>'));
      assert.ok(_.contains(xml, '<keyMaterial>p&amp;ss&lt;word&gt;</keyMaterial>'));
      assert.strictEqual(element(xml, 'authentication'), 'WPA2PSK');
      assert.strictEqual(element(xml, 'encryption'), 'AES');
    });

    it('writes WPA3 Personal, with transition mode inside authEncryption', function() {
      assert.strictEqual(element(wlan({ ssid: 'Home', security: 'WPA3Personal', passphrase: 'correct horse' }), 'authentication'), 'WPA3SAE');
      let transition = wlan({ ssid: 'Home', security: 'WPA3PersonalTransition', passphrase: 'correct horse' });
      assert.ok(_.contains(transition, '<useOneX>false</useOneX><transitionMode xmlns="http://www.microsoft.com/networking/WLAN/profile/v4">true</transitionMode></authEncryption>'));
    });

    it('writes an open network with no key, and rejects one given a password', function() {
      let xml = wlan({ ssid: 'Lobby', security: 'Open' });
      assert.strictEqual(element(xml, 'authentication'), 'open');
      assert.strictEqual(element(xml, 'encryption'), 'none');
      assert.ok(!_.contains(xml, '<sharedKey>'));
      assert.throws(()=>{ wlan({ ssid: 'Lobby', security: 'Open', passphrase: 'x' }); }, /An open network has no password/);
    });

    it('takes a passphrase Windows accepts, writes a 64-digit hex key as a WPA2 network key, and rejects anything else', function() {
      assert.strictEqual(element(wlan({ ssid: 'Home', security: 'WPA2Personal', passphrase: 'a'.repeat(63) }), 'keyType'), 'passPhrase');
      assert.strictEqual(element(wlan({ ssid: 'Home', security: 'WPA2Personal', passphrase: 'ab'.repeat(32) }), 'keyType'), 'networkKey');
      for (let security of ['WPA3Personal', 'WPA3PersonalTransition']) {
        assert.throws(()=>{ wlan({ ssid: 'Home', security, passphrase: 'ab'.repeat(32) }); }, /A WPA3 Wi-Fi password must be 8 to 63/, security);
      }
      for (let passphrase of ['wifi', 'z'.repeat(64), 'pässwört-123']) {
        assert.throws(()=>{ wlan({ ssid: 'Home', security: 'WPA2Personal', passphrase }); }, /8 to 63 printable ASCII characters/, passphrase);
      }
    });

    it('keeps server certificate validation on for PEAP, asking the user when nothing says what to trust', function() {
      let unpinned = wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'PEAP' } });
      assert.strictEqual(whyNotWellFormed(unpinned), undefined);
      assert.strictEqual(element(unpinned, 'DisableUserPromptForServerValidation'), 'false');
      assert.strictEqual(element(unpinned, 'PerformServerValidation'), 'true');
      assert.strictEqual(element(unpinned, 'AcceptServerName'), 'false');

      let pinned = wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'peap-mschapv2', serverNames: 'r1.example.com, r2.example.com', trustedRootCAThumbprints: 'SHA1: 3B 1E FD 3A 66 EA 28 B1 66 97 39 47 03 A7 2C A3 40 A0 5B D5' } });
      assert.strictEqual(element(pinned, 'DisableUserPromptForServerValidation'), 'true');
      assert.strictEqual(element(pinned, 'AcceptServerName'), 'true');
      assert.strictEqual(element(pinned, 'ServerNames'), 'r1.example.com;r2.example.com');
      assert.strictEqual(element(pinned, 'TrustedRootCA'), '3b 1e fd 3a 66 ea 28 b1 66 97 39 47 03 a7 2c a3 40 a0 5b d5');
      assert.ok(appearsInOrder(pinned, ['<authEncryption><authentication>WPA2</authentication><encryption>AES</encryption><useOneX>true</useOneX></authEncryption><OneX', '<EapMethod>', '<Type xmlns="http://www.microsoft.com/provisioning/EapCommon">25</Type>', '<ServerValidation>', '<FastReconnect>', '<Type>26</Type>', '<PeapExtensions>']));
    });

    it('keeps server certificate validation on for EAP-TLS, and accepts server names only when given', function() {
      let unpinned = wlan({ ssid: 'Corp', security: 'WPA3Enterprise', eap: { method: 'EAP-TLS' } });
      assert.strictEqual(whyNotWellFormed(unpinned), undefined);
      assert.strictEqual(element(unpinned, 'authentication'), 'WPA3ENT');
      assert.strictEqual(element(unpinned, 'PerformServerValidation'), 'true');
      assert.strictEqual(element(unpinned, 'AcceptServerName'), 'false');
      assert.strictEqual(element(unpinned, 'DisableUserPromptForServerValidation'), 'false');

      let rootOnly = wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'TLS', trustedRootCAThumbprints: ['3b1efd3a66ea28b16697394703a72ca340a05bd5'] } });
      assert.strictEqual(element(rootOnly, 'DisableUserPromptForServerValidation'), 'true');
      assert.strictEqual(element(rootOnly, 'AcceptServerName'), 'false');
      assert.strictEqual(element(rootOnly, 'TrustedRootCA'), '3b 1e fd 3a 66 ea 28 b1 66 97 39 47 03 a7 2c a3 40 a0 5b d5');
    });

    it('writes WPA3-Enterprise 192-bit mode with EAP-TLS only', function() {
      let xml = wlan({ ssid: 'Corp', security: 'WPA3Enterprise192', eap: { method: 'TLS' } });
      assert.strictEqual(element(xml, 'authentication'), 'WPA3ENT192');
      assert.strictEqual(element(xml, 'encryption'), 'GCMP256');
      assert.throws(()=>{ wlan({ ssid: 'Corp', security: 'WPA3Enterprise192', eap: { method: 'PEAP' } }); }, /only supports EAP-TLS/);
    });

    it('writes EAP-TTLS with MSCHAPv2 inside, prompting only when nothing says what to trust', function() {
      let unpinned = wlan({ ssid: 'Campus', security: 'WPA2Enterprise', eap: { method: 'TTLS' } });
      assert.strictEqual(whyNotWellFormed(unpinned), undefined);
      assert.ok(_.contains(unpinned, '<AuthorId xmlns="http://www.microsoft.com/provisioning/EapCommon">311</AuthorId>'));
      assert.ok(_.contains(unpinned, '<Phase2Authentication><MSCHAPv2Authentication>'));
      assert.strictEqual(element(unpinned, 'DisablePrompt'), 'false');
      let pinned = wlan({ ssid: 'Campus', security: 'WPA2Enterprise', eap: { method: 'EAP-TTLS', serverNames: ['radius.example.edu'], trustedRootCAThumbprints: ['3b1efd3a66ea28b16697394703a72ca340a05bd5'] } });
      assert.strictEqual(element(pinned, 'DisablePrompt'), 'true');
      assert.strictEqual(element(pinned, 'TrustedRootCAHash'), '3b 1e fd 3a 66 ea 28 b1 66 97 39 47 03 a7 2c a3 40 a0 5b d5');
    });

    it('uses only the http:// namespaces the schemas declare', function() {
      for (let method of ['PEAP', 'TLS', 'TTLS']) {
        let namespaces = wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method } }).match(/xmlns="[^"]+"/g);
        assert.ok(namespaces.length > 3);
        assert.ok(_.every(namespaces, (namespace)=>{ return /^xmlns="http:\/\/www\.microsoft\.com\//.test(namespace); }), namespaces.join(' '));
      }
    });

    it('rejects a thumbprint that is not 40 hex digits, and an EAP method it cannot write', function() {
      assert.throws(()=>{ wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'PEAP', trustedRootCAThumbprints: ['a1b2'] } }); }, /is not a root CA thumbprint/);
      assert.throws(()=>{ wlan({ ssid: 'Corp', security: 'WPA2Enterprise', eap: { method: 'EAP-SIM' } }); }, /needs an EAP method of PEAP, TLS, TTLS, not SIM/);
      assert.throws(()=>{ wlan({ ssid: 'Corp', security: 'WPA2Enterprise' }); }, /not none/);
    });

  });

});
