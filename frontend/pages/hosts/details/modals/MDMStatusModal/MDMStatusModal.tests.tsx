import React from "react";
import { noop } from "lodash";
import { render } from "@testing-library/react";
import { createMockRouter } from "test/test-utils";

import MDMStatusModal from "./MDMStatusModal";

describe("MDMStatusModal", () => {
  it("renders basic location info and Google Maps link when hostGeolocation is provided", () => {
    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={createMockRouter()}
        onExit={noop}
      />
    );
  });
});
