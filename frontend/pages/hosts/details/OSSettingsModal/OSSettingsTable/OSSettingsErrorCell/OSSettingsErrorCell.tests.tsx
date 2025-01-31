import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";

import OSSettingsErrorCell from "./OSSettingsErrorCell";

describe("OSSettingsErrorCell", () => {
  it("should render a formatted message for windows profiles", () => {
    render(
      <OSSettingsErrorCell
        canResendProfiles
        hostId={1}
        profile={createMockHostMdmProfile({
          platform: "windows",
          status: "failed",
          detail:
            "starting encryption: encrypt(C:): error code returned during encryption: -2147024809, error 2: This is another error",
        })}
      />
    );

    const firstErrorKey = screen.getByText(
      (content) => content === "starting encryption:"
    );
    const firstErrorValue = screen.getByText(
      (content) =>
        content ===
        "encrypt(C:): error code returned during encryption: -2147024809,"
    );

    // assert that the tooltip errors are rendered and the key is bolded
    expect(firstErrorKey).toBeInTheDocument();
    expect(firstErrorKey.tagName.toLowerCase()).toBe("b");
    expect(firstErrorValue).toBeInTheDocument();

    const secondErrorKey = screen.getByText(
      (content) => content === "error 2:"
    );
    const secondErrorValue = screen.getByText(
      (content) => content === "This is another error"
    );

    // assert the second error is rendered with the key bolded
    expect(secondErrorKey).toBeInTheDocument();
    expect(secondErrorKey.tagName.toLowerCase()).toBe("b");
    expect(secondErrorValue).toBeInTheDocument();
  });

  it("renders a default empty cell when the status is not failed", () => {
    render(
      <OSSettingsErrorCell
        canResendProfiles
        hostId={1}
        profile={createMockHostMdmProfile({})}
      />
    );

    expect(screen.getAllByText("---")[0]).toBeInTheDocument();
  });

  it("renders a resend button when canResendProfiles is true and profile is failed", () => {
    render(
      <OSSettingsErrorCell
        canResendProfiles
        hostId={1}
        profile={createMockHostMdmProfile({ status: "failed" })}
      />
    );

    expect(screen.getByRole("button", { name: "Resend" })).toBeInTheDocument();
  });

  it("renders a resend button when canResendProfiles is true and profile is verified", () => {
    render(
      <OSSettingsErrorCell
        canResendProfiles
        hostId={1}
        profile={createMockHostMdmProfile({ status: "verified" })}
      />
    );

    expect(screen.getByRole("button", { name: "Resend" })).toBeInTheDocument();
  });

  it("renders a tooltip link when the error message inlcudes info about IDP emails", () => {
    render(
      <OSSettingsErrorCell
        canResendProfiles
        hostId={1}
        profile={createMockHostMdmProfile({
          status: "failed",
          detail: "There is no IdP email for this host.",
        })}
      />
    );

    // couldnt get getByRole to work for this link. Thinking it may be a jest issue
    // TODO: explore why getByRole is not working for links
    expect(screen.getByText(/Learn more/)).toBeInTheDocument();
    expect(screen.getByText(/Learn more/).tagName.toLowerCase()).toBe("a");
  });
});
