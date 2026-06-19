import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";

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

  // For a wrapped form field, the tooltip anchors to the label/control row rather than the
  // whole field, so the arrow points at the label instead of the center of
  // label + input + help text (#44325). jsdom has no layout, so the geometric arrow position
  // is verified manually; here we assert the anchor binding and trigger behavior.
  it("anchors the tooltip to the field's control row, not the help text, when GOM is enabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    });

    const { user, container } = render(
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            name="deleteActivities"
            value={false}
            helpText="This setting will delete existing results."
          >
            Delete activities
          </Checkbox>
        )}
      />
    );

    // The anchor target (the checkbox's control row) and the help text are distinct
    // elements; the help text must never be the anchor.
    const controlRow = container.querySelector(".form-field > label");
    expect(controlRow).not.toBeNull();
    expect(
      container.querySelector(".form-field__help-text")
    ).toBeInTheDocument();

    // Hovering the help text (not an anchor) does not open the tooltip...
    await user.hover(screen.getByText(/will delete existing results/i));
    expect(screen.queryByRole("tooltip")).toBeNull();

    // ...but hovering the control row does.
    await user.hover(screen.getByText("Delete activities"));
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });
  });

  // When the wrapped content is a group (multiple fields/controls) rather than a single
  // field, keep whole-wrapper anchoring so hovering any control still shows the tooltip.
  it("keeps whole-wrapper anchoring for grouped content with multiple controls", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          isTeamAdmin: false,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "a.b.cc",
            },
          },
        },
      },
    });

    const { user } = render(
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <div>
            <Checkbox disabled={disableChildren} name="first" value={false}>
              First option
            </Checkbox>
            <Checkbox disabled={disableChildren} name="second" value={false}>
              Second option
            </Checkbox>
          </div>
        )}
      />
    );

    // Hovering the second control (not the first) still shows the tooltip.
    await user.hover(screen.getByText("Second option"));
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });
  });
});
