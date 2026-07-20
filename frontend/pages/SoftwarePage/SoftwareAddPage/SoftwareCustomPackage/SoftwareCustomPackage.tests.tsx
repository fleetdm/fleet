import React from "react";
import { render, screen } from "@testing-library/react";

import { GitOpsCustomPackageBanner } from "./SoftwareCustomPackage";

describe("GitOpsCustomPackageBanner", () => {
  it("renders the shared GitOps banner copy", () => {
    render(<GitOpsCustomPackageBanner />);
    expect(
      screen.getByText(/Add custom packages in GitOps mode/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /copy its SHA-256 hash into your YAML so the next GitOps workflow doesn.t delete it/i
      )
    ).toBeInTheDocument();
  });

  it("renders the YAML docs link pointing at learn-more-about/software-yaml", () => {
    render(<GitOpsCustomPackageBanner />);
    const link = screen.getByRole("link", { name: /YAML docs/i });
    expect(link).toHaveAttribute(
      "href",
      expect.stringMatching(/learn-more-about\/software-yaml$/)
    );
    expect(link).toHaveAttribute("target", "_blank");
  });
});
