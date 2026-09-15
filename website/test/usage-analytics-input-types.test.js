// A rejected input fails the whole request, not just that field, so every statistic in it is lost.
// Fleet servers of different versions report Fleet-maintained apps in different shapes and all of
// them keep posting here indefinitely.
const assert = require('assert');
const rttc = require('rttc');
const action = require('../api/controllers/webhooks/receive-usage-analytics');

const OLD_MAINTAINED_APPS = ['1password/darwin'];
const NEW_MAINTAINED_APPS = [{name: '1password/darwin', patchPolicy: true, softwareAutomation: false}];

const accepts = (type, value) => { try { rttc.validate(type, value); return true; } catch (unusedErr) { return false; } };

for (let inputName of ['fleetMaintainedAppsMacOS', 'fleetMaintainedAppsWindows']) {
  let type = action.inputs[inputName].type;
  assert(accepts(type, OLD_MAINTAINED_APPS), `${inputName} must accept an array of slug strings.`);
  assert(accepts(type, NEW_MAINTAINED_APPS), `${inputName} must accept an array of app objects.`);
  assert(!accepts(type, 'not-an-array'), `${inputName} must still reject a non-array.`);
}

assert(action.inputs.numPoliciesAutomationEnabledSoftware, 'numPoliciesAutomationEnabledSoftware must be declared, or it is silently discarded.');

console.log('✔  The usage analytics webhook accepts every reported Fleet-maintained app shape.');
