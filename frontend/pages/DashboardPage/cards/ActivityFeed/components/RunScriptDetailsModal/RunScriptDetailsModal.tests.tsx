import { render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { createMockScriptResult } from "__mocks__/scriptMock";
import scriptsAPI from "services/entities/scripts";

import RunScriptDetailsModal from "./RunScriptDetailsModal";

jest.mock("services/entities/scripts");

const mockScriptsAPI = scriptsAPI as jest.Mocked<typeof scriptsAPI>;

// Retries stay on, without the backoff, so the component's own predicate decides.
const renderModal = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { cacheTime: 0, retryDelay: 0 } },
  });
  return render(
    <QueryClientProvider client={client}>
      <RunScriptDetailsModal scriptExecutionId="abc" onCancel={jest.fn()} />
    </QueryClientProvider>
  );
};

describe("RunScriptDetailsModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("reports a cleaned-up result as gone, without retrying or offering the issue link", async () => {
    mockScriptsAPI.getScriptResult.mockRejectedValue({ status: 404 });

    renderModal();

    expect(
      await screen.findByText("These script results are no longer available.")
    ).toBeInTheDocument();
    expect(screen.queryByText(/file an issue/i)).not.toBeInTheDocument();
    expect(mockScriptsAPI.getScriptResult).toHaveBeenCalledTimes(1);
  });

  it("keeps the generic error and the issue link for other failures", async () => {
    mockScriptsAPI.getScriptResult.mockRejectedValue({ status: 500 });

    renderModal();

    expect(
      await screen.findByText("Close this modal and try again.")
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(mockScriptsAPI.getScriptResult).toHaveBeenCalledTimes(4)
    );
  });

  it("uses the configured timeout instead of a duration in script output", async () => {
    mockScriptsAPI.getScriptResult.mockResolvedValue(
      createMockScriptResult({
        exit_code: -1,
        message:
          "Timeout. Fleet stopped the script after 60 seconds to protect host performance.",
        output: "sleeping 180 seconds",
      })
    );

    const { container } = renderModal();

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
