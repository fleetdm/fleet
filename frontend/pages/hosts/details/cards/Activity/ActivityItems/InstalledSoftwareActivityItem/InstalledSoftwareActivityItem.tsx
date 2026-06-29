import React from "react";

import {
  getInstallUninstallStatusPredicate,
  getInstallUninstallStatusPredicatePassive,
  SCRIPT_PACKAGE_SOURCES,
} from "interfaces/software";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "installed-software-activity-item";

const InstalledSoftwareActivityItem = ({
  tab,
  activity,
  onShowDetails,
  onCancel,
  hideCancel,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const { actor_full_name: actorName, details } = activity;
  const {
    self_service,
    software_title: title,
    source,
    from_setup_experience,
  } = details;
  const status =
    details.status === "failed" ? "failed_uninstall" : details.status;
  const isScriptPackageSource = SCRIPT_PACKAGE_SOURCES.includes(source || "");

  // Self-service installs/uninstalls can be triggered by anyone who opens the
  // host's My device page, including admins. Drop the actor and switch to
  // passive voice so the activity reads "<software> was installed on this
  // host (self-service)." without misrepresenting who initiated it.
  if (self_service) {
    const passivePrefix = getInstallUninstallStatusPredicatePassive(
      status,
      isScriptPackageSource
    );
    return (
      <ActivityItem
        className={baseClass}
        activity={activity}
        hideCancel={hideCancel}
        onShowDetails={onShowDetails}
        onCancel={onCancel}
        isSoloActivity={isSoloActivity}
      >
        <b>{title}</b> {passivePrefix} on this host
        {from_setup_experience ? " during setup experience" : ""} (self-service)
        .
      </ActivityItem>
    );
  }

  let installedSoftwarePrefix = getInstallUninstallStatusPredicate(
    status,
    isScriptPackageSource
  );
  if (tab !== "past" && activity.fleet_initiated) {
    installedSoftwarePrefix =
      status === "pending_uninstall" ? "will uninstall" : "will install";
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
    >
      <b>{actorName ?? "Fleet"}</b> {installedSoftwarePrefix} <b>{title}</b> on
      this host
      {from_setup_experience ? " during setup experience" : ""}.
    </ActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
