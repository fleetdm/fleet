import { mount } from "enzyme";

import AppSettingsPage from "pages/admin/AppSettingsPage";
import { flatConfigStub } from "test/stubs";
import testHelpers from "test/helpers";

const { connectedComponent, reduxMockStore } = testHelpers;
const baseStore = {
  app: { config: flatConfigStub, enrollSecret: [] },
};

describe("AppSettingsPage - component", () => {
  it("renders", () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(connectedComponent(AppSettingsPage, { mockStore }));

    expect(page.find("AppSettingsPage").length).toEqual(1);
  });
});
