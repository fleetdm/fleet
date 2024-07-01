import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";
import Vpp from "./Vpp";

describe("Vpp Section", () => {
  it("render turn on apple mdm message when apple mdm is not turned on ", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: false }),
          }),
        },
      },
    });

    render(<Vpp router={createMockRouter()} />);

    expect(
      screen.getByRole("button", { name: "Turn on macOS MDM" })
    ).toBeInTheDocument();
  });

  it("render enable vpp when vpp is disabled", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            mdm: createMockMdmConfig({ enabled_and_configured: true }),
          }),
        },
      },
    });

    render(<Vpp router={createMockRouter()} />);

    expect(screen.getByRole("button", { name: "Enable" })).toBeInTheDocument();
  });

  // TODO: do this when integration with backend is done
  // it("render edit vpp when vpp is enabled", () => {
  //   const render = createCustomRenderer({
  //     context: {
  //       app: {
  //         config: createMockConfig({
  //           mdm: createMockMdmConfig({ enabled_and_configured: true }),
  //         }),
  //       },
  //     },
  //   });
  //   render(<Vpp router={createMockRouter()} />);
  //   expect(screen.getByRole("button", { name: "Enable" })).toBeInTheDocument();
  // });
});
