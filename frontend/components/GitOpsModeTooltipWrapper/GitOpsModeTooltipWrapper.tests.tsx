import React from "react";

import { noop } from "lodash";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import Button from "components/buttons/Button";

import GitOpsModeTooltipWrapper from "./GitOpsModeTooltipWrapper";

describe("GitOpsModeTooltipWrapper", () => {
  it("renders clickable children without a tooltip when GOM is not enabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          // thanks, DeepPartial!
          config: {
            gitops: {
              gitops_mode_enabled: false,
              repository_url: "",
            },
          },
        },
      },
    });

    const onSave = jest.fn();

    const { user } = render(
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button disabled={disableChildren} onClick={onSave}>
            Save
          </Button>
        )}
      />
    );

    const btn = screen.getByText("Save");
    expect(btn).toBeInTheDocument();

    await user.hover(btn);
    expect(screen.queryByRole("tooltip")).toBeNull();

    await user.click(btn);
    expect(onSave).toHaveBeenCalled();
  });

  it("renders non-clickable children with the tooltip when GOM is enabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          // thanks, DeepPartial!
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    });

    const onSave = jest.fn();

    const { user } = render(
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button disabled={disableChildren} onClick={onSave}>
            Save
          </Button>
        )}
      />
    );

    const btn = screen.getByText("Save");
    expect(btn).toBeInTheDocument();

    await user.hover(btn);
    expect(screen.getByRole("tooltip")).toBeInTheDocument();

    await user.click(btn);
    expect(onSave).not.toHaveBeenCalled();
  });
});
