import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import Smtp from "./Smtp";

describe("Smtp", () => {
  const mockHandleSubmit = jest.fn().mockResolvedValue(true);

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the STARTTLS guidance on hover of the SSL/TLS checkbox label", async () => {
    const mockConfig = createMockConfig();

    const { user } = renderWithSetup(
      <Smtp
        appConfig={mockConfig}
        handleSubmit={mockHandleSubmit}
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

    const label = screen.getByText("Use SSL/TLS to connect (recommended)");
    await user.hover(label);

    await waitFor(() => {
      expect(
        screen.getByText(/STARTTLS must first be disabled in/i)
      ).toBeInTheDocument();
    });
  });
});
