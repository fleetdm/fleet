import { render, screen } from "@testing-library/react";
import React from "react";

import EndUserOSRequirementPreview from "./EndUserOSRequirementPreview";

describe("EndUserOSRequirementPreview", () => {
  it.each(["ios", "ipados"] as const)(
    "shows the notification timing and the Apple learn more link for %s",
    (platform) => {
      render(<EndUserOSRequirementPreview platform={platform} />);

      expect(
        screen.getByText("End users are notified an hour before the deadline.")
      ).toBeVisible();
      expect(screen.getByRole("link", { name: /Learn more/ })).toHaveAttribute(
        "href",
        "https://fleetdm.com/learn-more-about/os-updates-apple"
      );
    }
  );
});
