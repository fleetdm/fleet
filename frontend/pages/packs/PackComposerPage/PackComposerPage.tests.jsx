import { mount } from "enzyme";

import { configStub } from "test/stubs";
import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedPacksComposerPage from "./PackComposerPage";

describe("PackComposerPage - component", () => {
  const mockStore = reduxMockStore({
    entities: {
      packs: {},
    },
    app: { config: configStub },
  });
  it("renders", () => {
    const page = mount(
      connectedComponent(ConnectedPacksComposerPage, { mockStore })
    );

    expect(page.length).toEqual(1);
  });

  it("renders a PackForm component", () => {
    const page = mount(
      connectedComponent(ConnectedPacksComposerPage, { mockStore })
    );

    expect(page.find("PackForm").length).toEqual(1);
  });

  it("renders a PackInfoSidePanel component", () => {
    const page = mount(
      connectedComponent(ConnectedPacksComposerPage, { mockStore })
    );

    expect(page.find("PackInfoSidePanel").length).toEqual(1);
  });
});
