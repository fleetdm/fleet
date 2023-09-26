import { IScriptResultResponse } from "services/entities/scripts";

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

const createMockScriptResult = (
  overrides?: Partial<IScriptResultResponse>
): IScriptResultResponse => {
  return { ...DEFAULT_SCRIPT_RESULT_MOCK, ...overrides };
};

export default createMockScriptResult;
