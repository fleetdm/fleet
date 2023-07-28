import React from "react";
import { render, screen } from "@testing-library/react";
import ClickableUrls from "./ClickableUrls";

const TEXT_WITH_URLS =
  "Contact your IT administrator to ensure your Mac is receiving a profile that disables advertisement tracking. https://privacyinternational.org/guide-step/4335/macos-opt-out-targeted-ads or https://support.apple.com/en-us/HT202074";
const URL_1 =
  "https://privacyinternational.org/guide-step/4335/macos-opt-out-targeted-ads";
const URL_2 = "https://support.apple.com/en-us/HT202074";

describe("ClickableUrls - component", () => {
  it("renders text and icon", () => {
    render(<ClickableUrls text={TEXT_WITH_URLS} />);

    const link1 = screen.getByRole("link", { name: URL_1 });
    const link2 = screen.getByRole("link", { name: URL_2 });

    expect(link1).toHaveAttribute("href", URL_1);
    expect(link1).toHaveAttribute("target", "_blank");
    expect(link2).toHaveAttribute("href", URL_2);
    expect(link2).toHaveAttribute("target", "_blank");
  });
});
