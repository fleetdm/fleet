import { screen } from "@testing-library/react";
import React from "react";

import { createMockScriptResult } from "__mocks__/scriptMock";
import scriptsAPI from "services/entities/scripts";
import { createCustomRenderer } from "test/test-utils";

import RunScriptDetailsModal from "./RunScriptDetailsModal";

jest.mock("services/entities/scripts");

describe("RunScriptDetailsModal", () => {
  it("uses the configured timeout instead of a duration in script output", async () => {
    (scriptsAPI.getScriptResult as jest.Mock).mockResolvedValue(
      createMockScriptResult({
        exit_code: -1,
        message:
          "Timeout. Fleet stopped the script after 60 seconds to protect host performance.",
        output: "sleeping 180 seconds",
      })
    );

    const { container } = createCustomRenderer({ withBackendMock: true })(
      <RunScriptDetailsModal scriptExecutionId="123" onCancel={jest.fn()} />
    );

    const statusMessage = await screen.findByText(/Timeout\./);
    expect(statusMessage.parentElement?.textContent).toContain(
      "after 60 seconds"
    );
    expect(statusMessage.parentElement?.textContent).not.toContain(
      "after 180 seconds"
    );
    expect(container.textContent).toContain("sleeping 180 seconds");
  });
});
