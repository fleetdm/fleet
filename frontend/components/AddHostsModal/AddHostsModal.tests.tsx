import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import AddHostsModal from "./AddHostsModal";

describe("Add hosts modal", () => {
  it("renders tabs", async () => {
    render(<AddHostsModal />);
  });

  it("renders tabs", async () => {
    render(<AddHostsModal />);
  });

  it("renders tabs", async () => {
    render(<AddHostsModal />);
  });

  it("renders tabs", async () => {
    render(<AddHostsModal />);
  });

  it("renders tabs", async () => {
    render(<AddHostsModal />);
  });

  it("renders global enroll secret", async () => {
    render(<AddHostsModal />);
  });

  it("renders team enroll secret if team selected", async () => {
    render(<AddHostsModal />);
  });

  it("renders sandbox mode", async () => {
    render(<AddHostsModal />);
  });
});
