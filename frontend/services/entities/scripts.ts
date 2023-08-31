import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  getScriptResult(id: number) {
    const { SCRIPT_RESULT } = endpoints;

    // TODO: uncomment when API is ready.
    // return sendRequest("GET", SCRIPT_RESULT(id));
    return new Promise((resolve) => {
      resolve({
        script_contents: "test contentsss here is here",
        exit_code: 0,
        output: "test output",
        message: "test message",
        runtime: 20,
      });
    });
  },
};
