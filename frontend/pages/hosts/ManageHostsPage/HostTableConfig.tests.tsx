import React from "react";
import { render } from "@testing-library/react";

import {
  generateAvailableTableHeaders,
  getPrimaryDeviceUser,
} from "./HostTableConfig";

describe("getPrimaryDeviceUser", () => {
  it("returns no primary email, no suffix, and no tooltip lines when there are no emails", () => {
    expect(getPrimaryDeviceUser([])).toEqual({
      primaryEmail: undefined,
      suffixCount: 0,
      tooltipLines: [],
    });
  });

  it("returns the single email as primary with no suffix and no tooltip lines when there is exactly one", () => {
    expect(
      getPrimaryDeviceUser([{ email: "solo@acmecorp.com", source: "custom" }])
    ).toEqual({
      primaryEmail: "solo@acmecorp.com",
      suffixCount: 0,
      tooltipLines: [],
    });
  });

  it("prioritizes the IdP-sourced email over chrome/custom, regardless of array order", () => {
    const { primaryEmail, suffixCount } = getPrimaryDeviceUser([
      { email: "custom1@acmecorp.com", source: "custom" },
      { email: "chrome@acmecorp.com", source: "google_chrome_profiles" },
      { email: "idp.user@acmecorp.com", source: "mdm_idp_accounts" },
      { email: "custom2@acmecorp.com", source: "custom" },
    ]);
    expect(primaryEmail).toBe("idp.user@acmecorp.com");
    expect(suffixCount).toBe(3);
  });

  it("falls back to the first chrome profile email when no IdP email is present", () => {
    const { primaryEmail, suffixCount } = getPrimaryDeviceUser([
      { email: "custom1@acmecorp.com", source: "custom" },
      { email: "chrome@acmecorp.com", source: "google_chrome_profiles" },
    ]);
    expect(primaryEmail).toBe("chrome@acmecorp.com");
    expect(suffixCount).toBe(1);
  });

  it("falls back to the first available email when neither IdP nor chrome sources are present", () => {
    const { primaryEmail, suffixCount } = getPrimaryDeviceUser([
      { email: "custom1@acmecorp.com", source: "custom" },
      { email: "custom2@acmecorp.com", source: "custom" },
    ]);
    expect(primaryEmail).toBe("custom1@acmecorp.com");
    expect(suffixCount).toBe(1);
  });

  it("lists the primary email first in tooltipLines, followed by the rest", () => {
    const { tooltipLines } = getPrimaryDeviceUser([
      { email: "custom1@acmecorp.com", source: "custom" },
      { email: "idp.user@acmecorp.com", source: "mdm_idp_accounts" },
    ]);
    expect(tooltipLines).toEqual([
      "idp.user@acmecorp.com",
      "custom1@acmecorp.com",
    ]);
  });

  it("caps tooltipLines at 5 entries, appending a '+N more' line for the remainder", () => {
    const users = Array.from({ length: 7 }, (_, i) => ({
      email: `user${i}@acmecorp.com`,
      source: "custom",
    }));
    const { suffixCount, tooltipLines } = getPrimaryDeviceUser(users);
    expect(suffixCount).toBe(6);
    expect(tooltipLines).toEqual([
      "user0@acmecorp.com",
      "user1@acmecorp.com",
      "user2@acmecorp.com",
      "user3@acmecorp.com",
      "user4@acmecorp.com",
      "+2 more",
    ]);
  });

  it("does not append a '+N more' line when the total is exactly the cap", () => {
    const users = Array.from({ length: 5 }, (_, i) => ({
      email: `user${i}@acmecorp.com`,
      source: "custom",
    }));
    const { tooltipLines } = getPrimaryDeviceUser(users);
    expect(tooltipLines).toHaveLength(5);
    expect(tooltipLines.some((line) => line.includes("more"))).toBe(false);
  });
});

describe("HostTableConfig - User email column", () => {
  const getDeviceMappingColumn = () => {
    const columns = generateAvailableTableHeaders({
      isFreeTier: false,
      isOnlyObserver: false,
    });
    const column = columns.find((c) => c.id === "device_mapping") as any;
    if (!column) throw new Error("device_mapping column not found");
    return column;
  };

  const renderCell = (value: Array<{ email: string; source: string }>) => {
    const Cell = getDeviceMappingColumn().Cell as React.ElementType;
    return render(<Cell cell={{ value }} />);
  };

  it("renders the default empty value with no suffix when there are no emails", () => {
    const { container } = renderCell([]);
    const textEl = container.querySelector(
      ".data-table__tooltip-truncated-text"
    );
    expect(textEl?.textContent).toBe("---");
    expect(
      container.querySelector(".data-table__suffix")
    ).not.toBeInTheDocument();
  });

  it("renders the primary email with a '+N' suffix and enables the tooltip when there are multiple emails", () => {
    const { container } = renderCell([
      { email: "custom1@acmecorp.com", source: "custom" },
      { email: "idp.user@acmecorp.com", source: "mdm_idp_accounts" },
    ]);
    const textEl = container.querySelector(
      ".data-table__tooltip-truncated-text"
    );
    const suffixEl = container.querySelector(".data-table__suffix");
    const tooltipTrigger = container.querySelector(
      ".data-table__tooltip-truncated-text-container"
    );

    expect(textEl?.textContent).toBe("idp.user@acmecorp.com");
    expect(suffixEl?.textContent).toBe("+1");
    // Tooltip should be enabled here even though the primary email isn't
    // long enough to be visually truncated, because a suffix is present.
    expect(tooltipTrigger?.getAttribute("data-tip-disable")).toBe("false");
  });

  it("does not render a suffix or force the tooltip open for a single, untruncated email", () => {
    const { container } = renderCell([
      { email: "solo@acmecorp.com", source: "custom" },
    ]);
    expect(
      container.querySelector(".data-table__suffix")
    ).not.toBeInTheDocument();
    const tooltipTrigger = container.querySelector(
      ".data-table__tooltip-truncated-text-container"
    );
    expect(tooltipTrigger?.getAttribute("data-tip-disable")).toBe("true");
  });
});
