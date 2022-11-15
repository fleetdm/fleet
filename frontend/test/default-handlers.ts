import { defaultActivityHandler } from "./handlers/activity-handlers";
import {
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
} from "./handlers/device-handler";

export const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const handlers = [
  defaultDeviceHandler,
  defaultDeviceMappingHandler,
  defaultActivityHandler,
];

export default handlers;
