import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import DeleteHostModal from "./DeleteHostModal";

describe("DeleteHostModal", () => {
  it("renders the number of hosts selected", () => {
    render(
      <DeleteHostModal
        selectedHostIds={[1, 2, 3]}
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
      />
    );
    expect(screen.getByText("3 hosts")).toBeVisible();
  });

  it("renders the host name when only the host name is provided", () => {
    render(
      <DeleteHostModal
        hostName="Host1"
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
      />
    );
    expect(screen.getByText("Host1")).toBeVisible();
  });

  it("renders the number of hosts selected with '+' after when select all matching hosts is true", () => {
    render(
      <DeleteHostModal
        selectedHostIds={[1, 2, 3]}
        hostsCount={50}
        isAllMatchingHostsSelected
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
      />
    );
    expect(screen.getByText("3+ hosts")).toBeVisible();
  });

  it("renders the host count with '+' and an additional warning when there are more than 500 hosts and select all matching hosts is true", () => {
    render(
      <DeleteHostModal
        selectedHostIds={[1, 2, 3]}
        hostsCount={500}
        isAllMatchingHostsSelected
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
      />
    );
    expect(screen.getByText("3+ hosts")).toBeVisible();
    expect(
      screen.getByText(
        "When deleting a large volume of hosts, it may take some time for this change to be reflected in the UI."
      )
    ).toBeVisible();
  });
});
