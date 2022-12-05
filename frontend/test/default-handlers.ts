import { defaultActivityHandler } from "./handlers/activity-handlers";
import {
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultMacAdminsHandler,
} from "./handlers/device-handler";

export const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const handlers = [
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultMacAdminsHandler,
  defaultActivityHandler,
];

export default handlers;
