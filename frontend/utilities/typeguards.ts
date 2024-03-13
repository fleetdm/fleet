import { IGlobalCalendarIntegration } from "interfaces/integration";

const isBoolean = (value: any): value is boolean => typeof value === "boolean";

const isGlobalCalendarConfig = (
  value: any
): value is IGlobalCalendarIntegration[] => {
  // if it's an array, it's the global config
  return value.length !== undefined;
};

export default { isBoolean, isGlobalCalendarConfig };
