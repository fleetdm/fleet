import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import { IMdmProfile } from "interfaces/mdm";

import ProfileLabelsModal from "./ProfileLabelsModal";

const render = createCustomRenderer();

const baseProfile: IMdmProfile = {
  profile_uuid: "abc-123",
  team_id: 0,
  name: "Test Profile",
  platform: "darwin",
  identifier: "com.example.test",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  checksum: null,
};

describe("ProfileLabelsModal", () => {
  it("renders null when profile is null", () => {
    const { container } = render(
      <ProfileLabelsModal profile={null} setModalData={noop as any} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the include any section with correct labels", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_include_any: [{ name: "Label A" }, { name: "Label B" }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.getByText("Include any")).toBeInTheDocument();
    expect(screen.getByText("Label A")).toBeInTheDocument();
    expect(screen.getByText("Label B")).toBeInTheDocument();
    expect(screen.queryByText("Exclude any")).not.toBeInTheDocument();
  });

  it("renders the include all section when labels_include_all is set", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_include_all: [{ name: "Label A" }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.getByText("Include all")).toBeInTheDocument();
  });

  it("renders the exclude any section with correct labels", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_exclude_any: [{ name: "Label C" }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.getByText("Exclude any")).toBeInTheDocument();
    expect(screen.getByText("Label C")).toBeInTheDocument();
    expect(screen.queryByText("Include any")).not.toBeInTheDocument();
  });

  it("renders both include and exclude sections when both are set", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_include_any: [{ name: "Label A" }],
          labels_exclude_any: [{ name: "Label B" }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.getByText("Include any")).toBeInTheDocument();
    expect(screen.getByText("Exclude any")).toBeInTheDocument();
    expect(screen.getByText("Label A")).toBeInTheDocument();
    expect(screen.getByText("Label B")).toBeInTheDocument();
  });

  it("shows the broken label warning when any label is broken", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_include_any: [{ name: "Label A", broken: true }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.getByText(/broken/)).toBeInTheDocument();
    expect(screen.getByText("Label deleted")).toBeInTheDocument();
  });

  it("does not show the broken label warning when no labels are broken", () => {
    render(
      <ProfileLabelsModal
        profile={{
          ...baseProfile,
          labels_include_any: [{ name: "Label A" }],
        }}
        setModalData={noop as any}
      />
    );

    expect(screen.queryByText("Label deleted")).not.toBeInTheDocument();
  });
});
