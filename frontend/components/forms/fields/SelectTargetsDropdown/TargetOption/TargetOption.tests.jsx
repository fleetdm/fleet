import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import { hostStub, labelStub } from "test/stubs";
import TargetOption from "./TargetOption";

describe("TargetOption - component", () => {
  const onMoreInfoClickSpy = jest.fn();
  const onMoreInfoClick = () => {
    return onMoreInfoClickSpy;
  };
  it("renders a label option for label targets", () => {
    const count = 5;
    const { container } = render(
      <TargetOption
        onMoreInfoClick={onMoreInfoClick}
        target={{ ...labelStub, count }}
      />
    );
    expect(container.querySelectorAll(".is-label").length).toEqual(1);
    expect(screen.getByText(`${count} hosts`)).toBeInTheDocument();
  });

  it("renders a host option for host targets", () => {
    const { container } = render(
      <TargetOption
        onMoreInfoClick={onMoreInfoClick}
        target={{ ...hostStub, platform: "windows" }}
      />
    );
    expect(container.querySelectorAll(".is-host").length).toEqual(1);
    expect(container.querySelectorAll("i.fleeticon-windows").length).toEqual(1);
    expect(screen.getByText(hostStub.primary_ip)).toBeInTheDocument();
  });

  it("calls the onSelect prop when + icon button is clicked", () => {
    const onSelectSpy = jest.fn();
    const { container } = render(
      <TargetOption
        onMoreInfoClick={onMoreInfoClick}
        onSelect={onSelectSpy}
        target={hostStub}
      />
    );
    fireEvent.click(container.querySelector(".target-option__add-btn"));
    expect(onSelectSpy).toHaveBeenCalled();
  });

  it("calls the onMoreInfoClick prop when the item content is clicked", () => {
    const { container } = render(
      <TargetOption onMoreInfoClick={onMoreInfoClick} target={hostStub} />
    );
    fireEvent.click(container.querySelector(".target-option__target-content"));
    expect(onMoreInfoClickSpy).toHaveBeenCalled();
  });
});
