import React from "react";
import { mount } from "enzyme";
import { hostStub } from "test/stubs";
import hostActions from "redux/nodes/entities/hosts/actions";
import { HostDetailsPage } from "./HostDetailsPage";

const offlineHost = { ...hostStub, id: 111, status: "offline" };
const onlineHost = { ...hostStub, id: 111, status: "online" };

describe("HostDetailsPage - component", () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  const propsWithOnlineHost = {
    host: onlineHost,
    hostID: onlineHost.id.toString(),
    queries: {},
  };

  const propsWithOfflineHost = {
    host: offlineHost,
    hostID: offlineHost.id.toString(),
  };

  describe("Loading host data", () => {
    it("Loads host data", () => {
      const dispatch = () => Promise.resolve();
      const props = { ...propsWithOnlineHost, dispatch };
      const spy = jest
        .spyOn(hostActions, "load")
        .mockImplementation(() => () => Promise.resolve([]));
      mount(<HostDetailsPage {...props} />);
      expect(spy).toHaveBeenCalled();
    });
  });

  describe("Delete a host", () => {
    it("Deletes an offine host after confirmation modal", () => {
      const dispatch = () => Promise.resolve();
      const props = { ...propsWithOfflineHost, dispatch };
      const page = mount(<HostDetailsPage {...props} />);
      const deleteBtn = page.find("Button").at(2);
      expect(deleteBtn.text()).toContain("Delete");

      jest.spyOn(hostActions, "destroy").mockImplementation(() => () => {
        dispatch({ type: "hosts_LOAD_REQUEST" });

        return Promise.resolve();
      });

      expect(page.find("Modal").length).toEqual(0);

      deleteBtn.simulate("click");

      const confirmModal = page.find("Modal");
      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find(".button--alert");
      confirmBtn.simulate("click");

      expect(hostActions.destroy).toHaveBeenCalledWith(offlineHost);
    });

    it("Deletes an online host after confirmation modal", () => {
      const dispatch = () => Promise.resolve();
      const props = { ...propsWithOnlineHost, dispatch };
      const page = mount(<HostDetailsPage {...props} />);
      const deleteBtn = page.find("Button").at(2);
      expect(deleteBtn.text()).toContain("Delete");

      jest.spyOn(hostActions, "destroy").mockImplementation(() => () => {
        dispatch({ type: "hosts_LOAD_REQUEST" });

        return Promise.resolve();
      });

      expect(page.find("Modal").length).toEqual(0);

      deleteBtn.simulate("click");

      const confirmModal = page.find("Modal");
      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find(".button--alert");
      confirmBtn.simulate("click");

      expect(hostActions.destroy).toHaveBeenCalledWith(onlineHost);
    });
  });

  describe("Query a host", () => {
    const dispatch = () => Promise.resolve();
    const props = { ...propsWithOnlineHost, dispatch };
    const page = mount(<HostDetailsPage {...props} />);
    const queryBtn = page.find("Button").at(1);
    expect(queryBtn.text()).toContain("Query");

    expect(page.find("Modal").length).toEqual(0);

    queryBtn.simulate("click");

    const confirmModal = page.find("Modal");
    console.log("confirmModal", confirmModal);

    expect(confirmModal.length).toEqual(1);
  });
});
