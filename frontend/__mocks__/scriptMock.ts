import { IScript, IScriptResultResponse } from "services/entities/scripts";

const DEFAULT_SCRIPT_MOCK: IScript = {
  id: 1,
  team_id: null,
  name: "test script",
  created_at: "2020-01-01T00:00:00.000Z",
  updated_at: "2020-01-01T00:00:00.000Z",
};

export const createMockScript = (overrides?: Partial<IScript>): IScript => {
  return { ...DEFAULT_SCRIPT_MOCK, ...overrides };
};

const DEFAULT_SCRIPT_RESULT_MOCK: IScriptResultResponse = {
  hostname: "Test Host",
  host_id: 1,
  execution_id: "123",
  script_contents: "ls /home/*\necho 'testing'\necho 'lines'\nexit $?",
  exit_code: 0,
  output: "test\nlines\n",
  message: "",
  runtime: 0,
  host_timeout: false,
};

export const createMockScriptResult = (
  overrides?: Partial<IScriptResultResponse>
): IScriptResultResponse => {
  return { ...DEFAULT_SCRIPT_RESULT_MOCK, ...overrides };
};
