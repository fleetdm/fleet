import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import TargetDetails from "components/forms/fields/SelectTargetsDropdown/TargetDetails";
import Test from "test";

describe("TargetDetails - component", () => {
  const defaultProps = { target: Test.Stubs.labelStub };

  describe("rendering", () => {
    it("does not render without a target", () => {
      const { container } = render(<TargetDetails />);

      expect(container).toBeEmptyDOMElement();
    });

    it("renders when there is a target", () => {
      const { container } = render(<TargetDetails {...defaultProps} />);

      expect(container).not.toBeEmptyDOMElement();
    });

    describe("when the target is a host", () => {
      it("renders target information", () => {
        const target = {
          display_text: "display_text",
          primary_mac: "host_mac",
          primary_ip: "host_ip_address",
          memory: 1074000000, // 1 GB memory
          osquery_version: "osquery_version",
          os_version: "os_version",
          platform: "platform",
          status: "status",
        };
        render(<TargetDetails target={target} />);

        expect(screen.getByText(target.display_text)).toBeInTheDocument();
        expect(screen.getByText(target.primary_mac)).toBeInTheDocument();
        expect(screen.getByText(target.primary_ip)).toBeInTheDocument();
        expect(screen.getByText("1.0 GB")).toBeInTheDocument();
        expect(screen.getByText(target.osquery_version)).toBeInTheDocument();
        expect(screen.getByText(target.os_version)).toBeInTheDocument();
        expect(screen.getByText(target.platform)).toBeInTheDocument();
        expect(screen.getByText(target.status)).toBeInTheDocument();
      });

      it("renders a success check icon when the target is online", () => {
        const target = { ...Test.Stubs.hostStub, status: "online" };
        const { container } = render(<TargetDetails target={target} />);
        const onlineIcon = container.querySelectorAll(
          ".host-target__icon--online"
        );
        const offlineIcon = container.querySelectorAll(
          ".host-target__icon--offline"
        );

        expect(onlineIcon.length).toBeGreaterThan(
          0,
          "Expected the online icon to render"
        );
        expect(offlineIcon.length).toEqual(
          0,
          "Expected the offline icon to not render"
        );
      });

      it("renders a offline icon when the target is offline", () => {
        const target = { ...Test.Stubs.hostStub, status: "offline" };
        const { container } = render(<TargetDetails target={target} />);
        const onlineIcon = container.querySelectorAll(
          ".host-target__icon--online"
        );
        const offlineIcon = container.querySelectorAll(
          ".host-target__icon--offline"
        );

        expect(onlineIcon.length).toEqual(
          0,
          "Expected the online icon to not render"
        );
        expect(offlineIcon.length).toBeGreaterThan(
          0,
          "Expected the offline icon to render"
        );
      });
    });

    describe("when the target is a label", () => {
      const target = {
        ...Test.Stubs.labelStub,
        count: 10,
        description: "target description",
        display_text: "display_text",
        label_type: 0,
        online: 10,
        query: "query",
      };

      it("renders the label data", () => {
        render(<TargetDetails target={target} />);
        expect(screen.getByText(/ONLINE/)).toBeInTheDocument();
        expect(screen.getByText(target.display_text)).toBeInTheDocument();
        expect(screen.getByText(target.description)).toBeInTheDocument();
      });

      it("renders a read-only AceEditor", () => {
        render(<TargetDetails target={target} />);
        expect(screen.getByRole("textbox")).toHaveAttribute("readonly");
      });
    });
  });

  it("calls the handleBackToResults prop when the back button is clicked", () => {
    const labelSpy = jest.fn();
    const labelProps = { ...defaultProps, handleBackToResults: labelSpy };
    const { rerender } = render(<TargetDetails {...labelProps} />);

    fireEvent.click(screen.getByRole("button"));

    expect(labelSpy).toHaveBeenCalled();

    const hostSpy = jest.fn();
    const hostProps = {
      target: Test.Stubs.hostStub,
      handleBackToResults: hostSpy,
    };

    rerender(<TargetDetails {...hostProps} />);

    fireEvent.click(screen.getByRole("button"));

    expect(hostSpy).toHaveBeenCalled();
  });
});
