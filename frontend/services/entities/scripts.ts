import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IScriptResult {
  host_name: string;
  host_id: number;
  execution_id: string;
  script_contents: string;
  exit_code: number | null;
  output: string;
  message: string;
  runtime: number;
  host_timeout: boolean;
}

export default {
  getScriptResult(id: string) {
    const { SCRIPT_RESULT } = endpoints;

    // TODO: uncomment when API is ready.
    // return sendRequest("GET", SCRIPT_RESULT(id));
    return new Promise<IScriptResult>((resolve) => {
      resolve({
        host_name: "test host",
        host_id: 1,
        execution_id: "test-id",
        script_contents: "test contentsss here is here",
        exit_code: 1,
        output: "test output",
        message: "Error: This is an error message.",
        runtime: 20,
        host_timeout: false,
      });
    });
  },
};
