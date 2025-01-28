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
const handlers = [
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultMacAdminsHandler,
  defaultActivityHandler,
];

export default handlers;
