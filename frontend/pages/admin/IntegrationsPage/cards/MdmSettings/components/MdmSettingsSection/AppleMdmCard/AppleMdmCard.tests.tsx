import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import createMockMdmApple from "__mocks__/appleMdm";
import createMockAxiosError from "__mocks__/axiosError";

import AppleMdmCard from "./AppleMdmCard";

describe("AppleMdmCard", () => {
  it("renders the turn on Apple mdm state when there is no appleAPNInfo", () => {
    render(
      <AppleMdmCard
        appleAPNSInfo={undefined}
        errorData={null}
        turnOnAppleMdm={noop}
        viewDetails={noop}
      />
    );

    expect(
      screen.getByText("Turn on Apple (macOS, iOS, iPadOS) MDM")
    ).toBeInTheDocument();
  });

  it("renders the show details state when there is appleAPNInfo", () => {
    render(
      <AppleMdmCard
        appleAPNSInfo={createMockMdmApple()}
        errorData={null}
        turnOnAppleMdm={noop}
        viewDetails={noop}
      />
    );

    expect(
      screen.getByText("Apple (macOS, iOS, iPadOS) MDM turned on.")
    ).toBeInTheDocument();
  });

  it("renders the error state when there is a non 404 error", () => {
    render(
      <AppleMdmCard
        appleAPNSInfo={createMockMdmApple()}
        errorData={createMockAxiosError({ status: 500 })}
        turnOnAppleMdm={noop}
        viewDetails={noop}
      />
    );

    expect(screen.getByText(/Something's gone wrong/)).toBeInTheDocument();

    render(
      <AppleMdmCard
        appleAPNSInfo={createMockMdmApple()}
        errorData={createMockAxiosError({ status: 404 })}
        turnOnAppleMdm={noop}
        viewDetails={noop}
      />
    );
  });
});
