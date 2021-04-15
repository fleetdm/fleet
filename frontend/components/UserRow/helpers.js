const userActionOptions = (isCurrentUser, user, invite) => {
  const inviteActions = [
    { disabled: false, label: "Revoke Invitation", value: "revert_invitation" },
  ];
  const userEnableAction = user.enabled
    ? {
        disabled: isCurrentUser,
        label: "Disable Account",
        value: "disable_account",
      }
    : { disabled: false, label: "Enable Account", value: "enable_account" };
  const userPromotionAction = user.admin
    ? { disabled: isCurrentUser, label: "Demote User", value: "demote_user" }
    : { disabled: false, label: "Promote User", value: "promote_user" };

  if (invite) return inviteActions;

  const result = [userEnableAction, userPromotionAction];
  if (!user.sso_enabled) {
    result.push({
      disabled: false,
      label: "Require Password Reset",
      value: "reset_password",
      helpText: "This will revoke all active Fleet API tokens for this user.",
    });
  }
  result.push({
    disabled: false,
    label: "Modify Details",
    value: "modify_details",
  });
  return result;
};

const userStatusLabel = (user, invite) => {
  if (invite) {
    return "Invited";
  }

  return user.enabled ? "Active" : "Disabled";
};

export default { userActionOptions, userStatusLabel };
