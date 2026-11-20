import { screen } from "@testing-library/react";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import UnlockModal from "./UnlockModal";

const MOCK_PROPS = {
  id: 7,
  platform: "windows",
  hostName: "host-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("UnlockModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  it.each(["ios", "ipados"])("renders the new %s copy", (platform) => {
    render(<UnlockModal {...MOCK_PROPS} platform={platform} />);

    expect(
      screen.getByText(
        "Unlock the host when it's returned to your organization."
      )
    ).toBeInTheDocument();
    expect(screen.queryByText(/Lost Mode/)).not.toBeInTheDocument();
  });

  it.each(["windows", "ubuntu"])("keeps the %s copy", (platform) => {
    render(<UnlockModal {...MOCK_PROPS} platform={platform} />);

    expect(
      screen.getByText(/Are you sure you're ready to unlock/)
    ).toBeInTheDocument();
    expect(screen.getByText("host-1")).toBeInTheDocument();
  });
});
