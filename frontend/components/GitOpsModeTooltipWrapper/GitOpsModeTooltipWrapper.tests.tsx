import React from "react";

import { screen, waitFor } from "@testing-library/react";
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
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });

    await user.click(btn);
    expect(onSave).not.toHaveBeenCalled();
  });

  it("renders clickable children when GOM is enabled but entity type is excepted", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
              exceptions: { labels: true, software: false, secrets: false },
            },
          },
        },
      },
    });

    const onSave = jest.fn();

    const { user } = render(
      <GitOpsModeTooltipWrapper
        entityType="labels"
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

  it("renders non-clickable children when GOM is enabled and entity type is not excepted", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
              exceptions: { labels: false, software: true, secrets: false },
            },
          },
        },
      },
    });

    const onSave = jest.fn();

    const { user } = render(
      <GitOpsModeTooltipWrapper
        entityType="labels"
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
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });

    await user.click(btn);
    expect(onSave).not.toHaveBeenCalled();
  });

  it("disables children when GOM is enabled and no entityType is specified", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
              exceptions: { labels: true, software: true, secrets: true },
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
    await user.click(btn);
    expect(onSave).not.toHaveBeenCalled();
  });
});
