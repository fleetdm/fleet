import React from "react";

import { createMockSoftwareVersion } from "__mocks__/softwareMock";
import { createCustomRenderer } from "test/test-utils";

import {
  SummarySection,
  getTitleDetailsPath,
} from "./SoftwareVersionDetailsPage";

// SoftwareIcon reads custom icons through react-query, so it needs that provider.
const render = createCustomRenderer({ withBackendMock: true });

describe("SummarySection", () => {
  it("links only the software name to the title when title_id is present", () => {
    const { container } = render(
      <SummarySection
        softwareVersion={createMockSoftwareVersion({ title_id: 7 })}
        hostsCount={3}
        teamIdForApi={1}
      />
    );

    const heading = container.querySelector("h1");
    const link = heading?.querySelector("a");
    expect(link).toBeTruthy();
    expect(link?.textContent).toContain("Test App");
    // the version stays outside the link, as plain heading text
    expect(link?.textContent).not.toContain("1.2.3");
    expect(heading?.textContent).toContain("1.2.3");
  });

  it("keeps the title heading plain text when title_id is missing", () => {
    const { container } = render(
      <SummarySection
        softwareVersion={createMockSoftwareVersion({ title_id: undefined })}
        hostsCount={3}
        teamIdForApi={1}
      />
    );

    const heading = container.querySelector("h1");
    expect(heading?.textContent).toBeTruthy();
    expect(heading?.querySelector("a")).toBeFalsy();
  });

  it("builds the title path preserving fleet context", () => {
    expect(getTitleDetailsPath(7, 1)).toContain("/software/titles/7");
    expect(getTitleDetailsPath(7, 1)).toContain("fleet_id=1");
  });
});
