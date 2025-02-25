import { IGetSetupExperienceScriptResponse } from "services/entities/mdm";

const DEFAULT_SETUP_EXPIERENCE_SCRIPT: IGetSetupExperienceScriptResponse = {
  id: 1,
  team_id: null,
  name: "Test Script.sh",
  created_at: "2021-01-01T00:00:00Z",
  updated_at: "2021-01-01T00:00:00Z",
};

// eslint-disable-next-line import/prefer-default-export
export const createMockSetupExperienceScript = (
  overrides?: Partial<IGetSetupExperienceScriptResponse>
): IGetSetupExperienceScriptResponse => {
  return { ...DEFAULT_SETUP_EXPIERENCE_SCRIPT, ...overrides };
};
