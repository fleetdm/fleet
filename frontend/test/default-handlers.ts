import { defaultActivityHandler } from "./handlers/activity-handlers";
import {
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultMacAdminsHandler,
} from "./handlers/device-handler";

export const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

// These are the default handlers that are used when testing the frontend. They
// are used to mock the responses from the Fleet API when running tests.
// These can be overridden in individual tests using the .use() method on the
// mock server within the desired test.
// More info on .use() here: https://mswjs.io/docs/api/setup-worker/use/

// NOTE: adding default handlers here is an anti-pattern we are moving away from.
// It is an anti-pattern because it makes it difficult to understand what
// handlers are being used in a test. The preferred way is to use the mockServer.use()
// method in the test file itself.
const handlers = [
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultMacAdminsHandler,
  defaultActivityHandler,
];

export default handlers;
