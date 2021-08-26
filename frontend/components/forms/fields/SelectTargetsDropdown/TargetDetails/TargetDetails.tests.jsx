import React from "react";
import { mount } from "enzyme";

import TargetDetails from "components/forms/fields/SelectTargetsDropdown/TargetDetails";
import Test from "test";

describe("TargetDetails - component", () => {
  const defaultProps = { target: Test.Stubs.labelStub };

  describe("rendering", () => {
    it("does not render without a target", () => {
      const Component = mount(<TargetDetails />);

      expect(Component.html()).toBeFalsy();
    });

    it("renders when there is a target", () => {
      const Component = mount(<TargetDetails {...defaultProps} />);

      expect(Component.length).toEqual(1);
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
        const Component = mount(<TargetDetails target={target} />);
        const componentText = Component.text();

        expect(componentText).toContain(target.display_text);
        expect(componentText).toContain(target.primary_mac);
        expect(componentText).toContain(target.primary_ip);
        expect(componentText).toContain("1.0 GB");
        expect(componentText).toContain(target.osquery_version);
        expect(componentText).toContain(target.os_version);
        expect(componentText).toContain(target.platform);
        expect(componentText).toContain(target.status);
      });

      it("renders a success check icon when the target is online", () => {
        const target = { ...Test.Stubs.hostStub, status: "online" };
        const Component = mount(<TargetDetails target={target} />);
        const FleetIcon = Component.find("FleetIcon");
        const onlineIcon = FleetIcon.find(".host-target__icon--online");
        const offlineIcon = FleetIcon.find(".host-target__icon--offline");

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
        const Component = mount(<TargetDetails target={target} />);
        const FleetIcon = Component.find("FleetIcon");
        const onlineIcon = FleetIcon.find(".host-target__icon--online");
        const offlineIcon = FleetIcon.find(".host-target__icon--offline");

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
      const Component = mount(<TargetDetails target={target} />);

      it("renders the label data", () => {
        const componentText = Component.text();

        expect(componentText).toContain("100% ONLINE");
        expect(componentText).toContain(target.display_text);
        expect(componentText).toContain(target.description);
      });

      it("renders a read-only AceEditor", () => {
        const AceEditor = Component.find("ReactAce");

        expect(AceEditor.prop("readOnly")).toEqual(true);
      });
    });
  });

  it("calls the handleBackToResults prop when the back button is clicked", () => {
    const labelSpy = jest.fn();
    const labelProps = { ...defaultProps, handleBackToResults: labelSpy };
    const LabelComponent = mount(<TargetDetails {...labelProps} />);
    const LabelBackButton = LabelComponent.find(".label-target__back");

    const hostSpy = jest.fn();
    const hostProps = {
      target: Test.Stubs.hostStub,
      handleBackToResults: hostSpy,
    };
    const HostComponent = mount(<TargetDetails {...hostProps} />);
    const HostBackButton = HostComponent.find(".host-target__back");

    LabelBackButton.simulate("click");

    expect(labelSpy).toHaveBeenCalled();

    HostBackButton.simulate("click");

    expect(hostSpy).toHaveBeenCalled();
  });
});
