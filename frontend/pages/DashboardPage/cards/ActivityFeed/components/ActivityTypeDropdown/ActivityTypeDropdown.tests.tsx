import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import ActivityTypeDropdown from "./ActivityTypeDropdown";

describe("ActivityTypeDropdown", () => {
  it("lists the viewed unlock PIN activity type", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<ActivityTypeDropdown value="" onSelect={noop} />);

    await user.click(screen.getByText("All types"));

    expect(screen.getByText("Viewed unlock PIN")).toBeInTheDocument();
  });
});
