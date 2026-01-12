// TransferHostModal.test.tsx
import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import paths from "router/paths";

import TransferHostModal from "./TransferHostModal";

const teams = [
  { id: 1, name: "Team Alpha" },
  { id: 2, name: "Team Beta" },
];

const setup = (
  props: Partial<React.ComponentProps<typeof TransferHostModal>> = {}
) => {
  const onSubmit = jest.fn();
  const onCancel = jest.fn();

  const user = userEvent.setup();

  render(
    <TransferHostModal
      isGlobalAdmin={false}
      teams={teams as any}
      onSubmit={onSubmit}
      onCancel={onCancel}
      isUpdating={false}
      multipleHosts={false}
      hostsTeamId={1}
      {...props}
    />
  );

  return { user, onSubmit, onCancel };
};

describe("TransferHostModal", () => {
  it("renders title for single host and label", () => {
    setup();

    expect(screen.getByText("Transfer host")).toBeInTheDocument();

    expect(
      screen.getByText("Transfer host to:", { selector: "label" })
    ).toBeInTheDocument();
  });

  it("pluralizes title and label when multipleHosts is true", () => {
    setup({ multipleHosts: true });

    expect(screen.getByText("Transfer hosts")).toBeInTheDocument();

    expect(
      screen.getByText("Transfer selected hosts to:", { selector: "label" })
    ).toBeInTheDocument();
  });

  it("shows Create a team link when user is global admin", () => {
    setup({ isGlobalAdmin: true });

    expect(screen.getByText(/Create a team/i)).toBeInTheDocument();
  });

  it("does not show Create a team link when not global admin", () => {
    setup({ isGlobalAdmin: false });

    expect(screen.queryByText(/Create a team/i)).not.toBeInTheDocument();
  });

  it("disables Transfer button until a team is selected", () => {
    setup();

    const transferButton = screen.getByRole("button", { name: "Transfer" });
    expect(transferButton).toBeDisabled();
  });

  it("filters out current host team from dropdown options and includes No team when allowed", async () => {
    const { user } = setup({ hostsTeamId: 1 });

    const dropdown = screen.getByText(/Select a team/i);

    await user.click(dropdown);

    expect(screen.getByText("No team")).toBeInTheDocument();
    expect(screen.getByText("Team Beta")).toBeInTheDocument();
    expect(screen.queryByText("Team Alpha")).not.toBeInTheDocument();
  });

  it("does not allow transferring to No team again when host is already on no team", async () => {
    const { user } = setup({ hostsTeamId: 0 });

    const dropdown = screen.getByText(/Select a team/i);
    await user.click(dropdown);

    expect(screen.queryByText("No team")).not.toBeInTheDocument();
    expect(screen.getByText("Team Alpha")).toBeInTheDocument();
    expect(screen.getByText("Team Beta")).toBeInTheDocument();
  });

  it("enables Transfer after selecting a team and calls onSubmit with selected team", async () => {
    const { user, onSubmit } = setup({ hostsTeamId: 1 });

    const dropdown = screen.getByText(/Select a team/i);

    // Custom dropdown path: open then click an option
    await user.click(dropdown);
    await user.click(await screen.findByText("Team Beta"));

    const transferButton = screen.getByRole("button", { name: "Transfer" });
    expect(transferButton).not.toBeDisabled();

    await user.click(transferButton);

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("supports selecting No team and passes through the no-team object shape", async () => {
    const { user, onSubmit } = setup({ hostsTeamId: 1 });

    const dropdown = screen.getByText(/Select a team/i);

    await user.click(dropdown);
    const noTeamOption = await screen.findByRole("option", {
      name: "No team",
    });
    await user.click(noTeamOption);

    const transferButton = screen.getByRole("button", { name: "Transfer" });
    expect(transferButton).not.toBeDisabled();

    await user.click(transferButton);

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when Cancel button is clicked", async () => {
    const { user, onCancel } = setup();

    const cancelButton = screen.getByRole("button", { name: "Cancel" });
    await user.click(cancelButton);

    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
