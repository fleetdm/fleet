import { mount } from "enzyme";

import helpers from "test/helpers";
import { userStub } from "test/stubs";
import CoreLayout from "./CoreLayout";

const { connectedComponent, reduxMockStore } = helpers;

describe("CoreLayout - layouts", () => {
  const store = {
    app: { config: {} },
    auth: { user: userStub },
    notifications: {},
    persistentFlash: {
      showFlash: false,
      message: "",
    },
  };
  const mockStore = reduxMockStore(store);

  it("renders the FlashMessage component when notifications are present", () => {
    const storeWithNotifications = {
      app: { config: {} },
      auth: {
        user: userStub,
      },
      notifications: {
        alertType: "success",
        isVisible: true,
        message: "nice jerb!",
      },
      persistentFlash: {
        showFlash: false,
        message: "",
      },
    };
    const mockStoreWithNotifications = reduxMockStore(storeWithNotifications);
    const componentWithFlash = connectedComponent(CoreLayout, {
      mockStore: mockStoreWithNotifications,
    });
    const componentWithoutFlash = connectedComponent(CoreLayout, {
      mockStore,
    });

    const appWithFlash = mount(componentWithFlash);
    const appWithoutFlash = mount(componentWithoutFlash);

    expect(appWithFlash.length).toEqual(1);
    expect(appWithoutFlash.length).toEqual(1);

    expect(appWithFlash.find("FlashMessage").html()).toBeTruthy();
    expect(appWithoutFlash.find("FlashMessage").html()).toBeFalsy();
  });

  it("renders the PersistentFlash component when showFlash is true", () => {
    const storeWithPersistentFlash = {
      ...store,
      persistentFlash: {
        showFlash: true,
        message: "This is the flash message",
      },
    };

    const mockStoreWithPersistentFlash = reduxMockStore(
      storeWithPersistentFlash
    );

    const Layout = connectedComponent(CoreLayout, {
      mockStore: mockStoreWithPersistentFlash,
    });
    const MountedLayout = mount(Layout);

    expect(MountedLayout.find("PersistentFlash").length).toEqual(
      1,
      "Expected the Persistent Flash to be on the page"
    );
  });
});
