import { render } from "@testing-library/react";

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
    const { container } = render(
      connectedComponent(ConnectedPacksComposerPage, { mockStore })
    );

    expect(container).not.toBeNull();
  });

  it("renders a PackInfoSidePanel component", () => {
    const { container } = render(
      connectedComponent(ConnectedPacksComposerPage, { mockStore })
    );

    expect(container.querySelectorAll(".pack-info-side-panel").length).toEqual(
      1
    );
  });
});
