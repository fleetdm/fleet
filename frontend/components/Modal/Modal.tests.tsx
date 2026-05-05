import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import Modal from "./Modal";

describe("Modal", () => {
  it("renders title", () => {
    render(
      <Modal title="Foobar" onExit={noop}>
        <div>test</div>
      </Modal>
    );

    expect(screen.getByText("Foobar")).toBeVisible();
  });
});
