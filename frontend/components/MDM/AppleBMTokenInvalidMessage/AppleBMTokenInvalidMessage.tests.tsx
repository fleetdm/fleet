import React from "react";
import { render, screen } from "@testing-library/react";

import AppleBMTokenInvalidMessage from "./AppleBMTokenInvalidMessage";

describe("AppleBMTokenInvalidMessage", () => {
  it("renders singular copy for a single org name", () => {
    render(<AppleBMTokenInvalidMessage orgNames={["Acme Inc."]} />);

    expect(
      screen.getByText(
        "Your Apple Business (AB) token for Acme Inc. is invalid. macOS, iOS, and iPadOS hosts won’t automatically enroll into Fleet. Users with the admin role in Fleet can renew the token."
      )
    ).toBeInTheDocument();
  });

  it("joins two org names with 'and' and uses plural copy", () => {
    render(
      <AppleBMTokenInvalidMessage orgNames={["Acme Inc.", "Globex Corp."]} />
    );

    expect(
      screen.getByText(
        "Your Apple Business (AB) tokens for Acme Inc. and Globex Corp. are invalid. macOS, iOS, and iPadOS hosts won’t automatically enroll into Fleet. Users with the admin role in Fleet can renew the tokens."
      )
    ).toBeInTheDocument();
  });

  it("joins three or more org names with an Oxford comma", () => {
    render(
      <AppleBMTokenInvalidMessage
        orgNames={["Acme Inc.", "Globex Corp.", "Initech"]}
      />
    );

    expect(
      screen.getByText(
        "Your Apple Business (AB) tokens for Acme Inc., Globex Corp., and Initech are invalid. macOS, iOS, and iPadOS hosts won’t automatically enroll into Fleet. Users with the admin role in Fleet can renew the tokens."
      )
    ).toBeInTheDocument();
  });
});
