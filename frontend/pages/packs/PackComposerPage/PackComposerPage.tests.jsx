import { mount } from "enzyme";

import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedPacksComposerPage from "./PackComposerPage";

describe("PackComposerPage - component", () => {
  const mockStore = reduxMockStore({
    entities: {
      packs: {},
    },
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
