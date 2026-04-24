import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import softwareAPI from "services/entities/software";

import { NotificationContext } from "context/notification";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "software-homebrew";

interface ISoftwareHomebrewProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareHomebrew = ({ currentTeamId, router }: ISoftwareHomebrewProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [packageName, setPackageName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onSubmit = useCallback(async () => {
    const token = packageName.trim().toLowerCase();
    if (!token) {
      renderFlash("error", "Homebrew package name is required.");
      return;
    }

    setIsSubmitting(true);
    try {
      await softwareAPI.addHomebrewPackage({ token, teamId: currentTeamId });
      renderFlash("success", "Software added via Homebrew.");
      router.push(
        getPathWithQueryParams(PATHS.SOFTWARE_TITLES, {
          fleet_id: currentTeamId,
        })
      );
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Failed to add package.";
      renderFlash("error", message);
    } finally {
      setIsSubmitting(false);
    }
  }, [packageName, currentTeamId, router, renderFlash]);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__form`}>
        <InputField
          autofocus
          label="Homebrew package name"
          placeholder="e.g. firefox, slack, zoom"
          value={packageName}
          onChange={setPackageName}
          name="homebrewPackageName"
          inputOptions={{
            onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => {
              if (e.key === "Enter" && packageName.trim() && !isSubmitting) {
                e.preventDefault();
                onSubmit();
              }
            },
          }}
        />
        <Button
          className={`${baseClass}__submit`}
          disabled={isSubmitting || !packageName.trim()}
          onClick={onSubmit}
          isLoading={isSubmitting}
        >
          Add package
        </Button>
      </div>
    </div>
  );
};

export default SoftwareHomebrew;
