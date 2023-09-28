import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import createMockMdmApple from "__mocks__/appleMdm";
import createMockAxiosError from "__mocks__/axiosError";

import MacOSMdmCard from "./MacOSMdmCard";

describe("MacOSMdmCard", () => {
  it("renders the turn on macOs mdm state when there is no appleAPNInfo", () => {
    render(
      <MacOSMdmCard
        appleAPNInfo={undefined}
        errorData={null}
        turnOnMacOSMdm={noop}
        viewDetails={noop}
      />
    );

    expect(screen.getByText("Turn on macOS MDM")).toBeInTheDocument();
  });

  it("renders the show details state when there is appleAPNInfo", () => {
    render(
      <MacOSMdmCard
        appleAPNInfo={createMockMdmApple()}
        errorData={null}
        turnOnMacOSMdm={noop}
        viewDetails={noop}
      />
    );

    expect(screen.getByText("macOS MDM turned on")).toBeInTheDocument();
  });

  it("renders the error state when there is a non 404 error", () => {
    render(
      <MacOSMdmCard
        appleAPNInfo={createMockMdmApple()}
        errorData={createMockAxiosError({ status: 500 })}
        turnOnMacOSMdm={noop}
        viewDetails={noop}
      />
    );

    expect(screen.getByText(/Something's gone wrong/)).toBeInTheDocument();

    render(
      <MacOSMdmCard
        appleAPNInfo={createMockMdmApple()}
        errorData={createMockAxiosError({ status: 404 })}
        turnOnMacOSMdm={noop}
        viewDetails={noop}
      />
    );
  });
});
