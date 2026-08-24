import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { noop } from "lodash";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";

import {
  SEVERITY_RANGE_INVALID_MSG,
  SeverityValue,
} from "components/SeverityFilter";

import ChartFilterModal, {
  IChartFilterState,
  PLATFORM_OPTIONS,
} from "./ChartFilterModal";

describe("ChartFilterModal PLATFORM_OPTIONS", () => {
  it("offers mobile platforms (iOS, iPadOS, Android) alongside desktop", () => {
    const values = PLATFORM_OPTIONS.map((o) => o.value);
    expect(values).toEqual([
      "darwin",
      "windows",
      "linux",
      "chrome",
      "ios",
      "ipados",
      "android",
    ]);
  });

  it("labels the mobile platforms for display", () => {
    const labelFor = (value: string) =>
      PLATFORM_OPTIONS.find((o) => o.value === value)?.label;
    expect(labelFor("ios")).toBe("iOS");
    expect(labelFor("ipados")).toBe("iPadOS");
    expect(labelFor("android")).toBe("Android");
  });
});

describe("ChartFilterModal severity", () => {
  const baseFilters: IChartFilterState = {
    labelIDs: [],
    platforms: [],
    hostFilterMode: "none",
    selectedHosts: [],
    softwareFilters: [...ALL_CVE_SOFTWARE_CATEGORY_VALUES],
    knownExploit: false,
    epssMin: "",
    epssMax: "",
    // A preset carries its own bounds — see IChartFilterState.
    severity: "critical",
    cvssMin: "9",
    cvssMax: "10",
    excludeCVEs: [],
  };

  beforeEach(() => {
    mockServer.use(
      http.get(baseUrl("/hosts"), () =>
        HttpResponse.json({ hosts: [], software: null })
      ),
      http.get(baseUrl("/labels/summary"), () =>
        HttpResponse.json({ labels: [] })
      ),
      http.get(baseUrl("/vulnerabilities"), () =>
        HttpResponse.json({
          count: 0,
          counts_updated_at: "",
          vulnerabilities: [],
          meta: { has_next_results: false, has_previous_results: false },
        })
      )
    );
  });

  const renderModal = (
    props: Partial<React.ComponentProps<typeof ChartFilterModal>> = {}
  ) =>
    createCustomRenderer({ withBackendMock: true })(
      <ChartFilterModal
        filters={baseFilters}
        metric="cve"
        initialTab="software"
        onApply={noop}
        onCancel={noop}
        {...props}
      />
    );

  it("applies the current severity selection", async () => {
    const onApply = jest.fn();
    const { user } = renderModal({ onApply });

    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({
        severity: "critical",
        cvssMin: "9",
        cvssMax: "10",
      })
    );
  });

  it("resets severity to Any on Clear all and hides the button", async () => {
    const { user } = renderModal();

    // The Critical default is an active filter, so Clear all is offered.
    const clearAll = screen.getByRole("button", { name: /Clear all/i });
    await user.click(clearAll);

    // The severity control lives behind the Advanced options reveal.
    await user.click(screen.getByRole("button", { name: /Advanced options/i }));
    expect(screen.getByText("Any severity")).toBeInTheDocument();

    // Nothing is filtered anymore, so there is nothing left to clear.
    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: /Clear all/i })
      ).not.toBeInTheDocument();
    });
  });

  it("offers no Clear all for a Custom range with no bounds entered", () => {
    renderModal({
      filters: { ...baseFilters, severity: "custom", cvssMin: "", cvssMax: "" },
    });

    // Custom without bounds sends nothing to the API, so it must not count as
    // an active filter — otherwise Clear all would appear over unfiltered data.
    expect(
      screen.queryByRole("button", { name: /Clear all/i })
    ).not.toBeInTheDocument();
  });

  it("clears to Any severity, which applies with no bounds", async () => {
    const onApply = jest.fn();
    const { user } = renderModal({ onApply });

    await user.click(screen.getByRole("button", { name: /Clear all/i }));
    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ severity: "any", cvssMin: "", cvssMax: "" })
    );
  });

  // Per frontend/docs/patterns.md#data-validation: Apply stays enabled, errors
  // appear on blur of a dirty field or on a submit attempt, and clear on focus.
  describe("validation lifecycle", () => {
    const invalidCustom = {
      ...baseFilters,
      severity: "custom" as SeverityValue,
      cvssMin: "9",
      cvssMax: "3",
    };

    // Once an error replaces the label, getByLabelText can no longer find the
    // field, so address the score inputs by name.
    const maxScoreInput = () =>
      document.querySelector('input[name="maxScore"]') as HTMLInputElement;

    // Retypes Max so it stays inverted against Min 9, marking the field dirty.
    const editMaxToInvalid = async (
      user: ReturnType<typeof renderModal>["user"]
    ) => {
      await user.clear(maxScoreInput());
      await user.type(maxScoreInput(), "1");
    };

    it("keeps Apply enabled with invalid input", () => {
      renderModal({ filters: invalidCustom });

      expect(screen.getByRole("button", { name: /Apply/i })).toBeEnabled();
    });

    it("shows no error for a seeded invalid value until it is touched", async () => {
      const { user } = renderModal({ filters: invalidCustom });

      await user.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );

      // The range is inverted, but the user has not edited it — a GitOps-seeded
      // value must not greet them with an error.
      expect(
        screen.queryByText(SEVERITY_RANGE_INVALID_MSG)
      ).not.toBeInTheDocument();

      // Tabbing through without editing is still not "dirty".
      await user.click(screen.getByLabelText(/Max score/i));
      await user.tab();
      expect(
        screen.queryByText(SEVERITY_RANGE_INVALID_MSG)
      ).not.toBeInTheDocument();
    });

    it("surfaces the error on blur once the field is dirty, and clears it on focus", async () => {
      const { user } = renderModal({ filters: invalidCustom });

      await user.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );

      await editMaxToInvalid(user);
      // Still nothing while typing.
      expect(
        screen.queryByText(SEVERITY_RANGE_INVALID_MSG)
      ).not.toBeInTheDocument();

      await user.tab();
      expect(screen.getByText(SEVERITY_RANGE_INVALID_MSG)).toBeInTheDocument();

      // Focusing the field again restores its label immediately, without
      // waiting for a valid value.
      await user.click(maxScoreInput());
      expect(
        screen.queryByText(SEVERITY_RANGE_INVALID_MSG)
      ).not.toBeInTheDocument();
      expect(screen.getByLabelText(/Max score/i)).toBeInTheDocument();
    });

    it("shows every error and does not apply when Apply is clicked with invalid input", async () => {
      const onApply = jest.fn();
      const { user } = renderModal({ filters: invalidCustom, onApply });

      await user.click(screen.getByRole("button", { name: /^Apply$/i }));

      expect(onApply).not.toHaveBeenCalled();
      // Submit bypasses the dirty gate, and the field it names is reachable:
      // Advanced options opens itself rather than hiding the error.
      expect(screen.getByText(SEVERITY_RANGE_INVALID_MSG)).toBeInTheDocument();
    });

    it("switches to the Software tab so a submit-surfaced error is visible", async () => {
      const { user } = renderModal({
        filters: invalidCustom,
        initialTab: "hosts",
      });

      await user.click(screen.getByRole("button", { name: /^Apply$/i }));

      expect(screen.getByText(SEVERITY_RANGE_INVALID_MSG)).toBeInTheDocument();
    });

    it("clears a score error when the severity option changes", async () => {
      const { user } = renderModal({ filters: invalidCustom });

      await user.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );
      await editMaxToInvalid(user);
      await user.tab();
      expect(screen.getByText(SEVERITY_RANGE_INVALID_MSG)).toBeInTheDocument();

      // High severity rewrites both bounds, so the error describes nothing.
      await user.click(screen.getByRole("combobox", { name: "Severity" }));
      const high = screen
        .getAllByTestId("dropdown-option")
        .find((el) => el.textContent?.startsWith("High severity"));
      if (!high) {
        throw new Error("No High severity option");
      }
      await user.click(high);

      expect(
        screen.queryByText(SEVERITY_RANGE_INVALID_MSG)
      ).not.toBeInTheDocument();
    });

    it("submits on Enter from a score input", async () => {
      const onApply = jest.fn();
      const { user } = renderModal({ onApply });

      await user.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );
      await user.click(screen.getByRole("combobox", { name: "Severity" }));
      const custom = screen
        .getAllByTestId("dropdown-option")
        .find((el) => el.textContent?.startsWith("Custom severity"));
      if (!custom) {
        throw new Error("No Custom severity option");
      }
      await user.click(custom);

      await user.type(maxScoreInput(), "{Enter}");

      expect(onApply).toHaveBeenCalled();
    });

    it("does not submit on Enter from a search field", async () => {
      const onApply = jest.fn();
      const { user } = renderModal({ onApply, initialTab: "hosts" });

      // Enter mid-search would apply the filters and close the modal, which is
      // not what "press Enter after typing a search" should do.
      await user.type(
        screen.getByPlaceholderText(/Search name, hostname/i),
        "web{Enter}"
      );

      expect(onApply).not.toHaveBeenCalled();
    });

    it("applies once the input is valid", async () => {
      const onApply = jest.fn();
      const { user } = renderModal({ filters: invalidCustom, onApply });

      await user.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );
      await user.clear(maxScoreInput());
      await user.type(maxScoreInput(), "10");
      await user.click(screen.getByRole("button", { name: /^Apply$/i }));

      expect(onApply).toHaveBeenCalledWith(
        expect.objectContaining({
          severity: "custom",
          cvssMin: "9",
          cvssMax: "10",
        })
      );
    });
  });
});
