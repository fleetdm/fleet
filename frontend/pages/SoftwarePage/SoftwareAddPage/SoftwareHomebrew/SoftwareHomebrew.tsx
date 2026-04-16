import React, { useCallback, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import softwareAPI from "services/entities/software";
import Button from "components/buttons/Button";

const baseClass = "software-homebrew";

interface ISoftwareHomebrewProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareHomebrew = ({ currentTeamId, router }: ISoftwareHomebrewProps) => {
  const [packageName, setPackageName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");

  const onSubmit = useCallback(async () => {
    if (!packageName.trim()) {
      setError("Package name is required");
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      await softwareAPI.addHomebrewPackage({
        token: packageName.trim().toLowerCase(),
        teamId: currentTeamId,
      });

      window.location.href = getPathWithQueryParams(PATHS.SOFTWARE_TITLES, {
        fleet_id: currentTeamId,
      });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to add package");
    } finally {
      setIsSubmitting(false);
    }
  }, [packageName, currentTeamId, router]);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__form`}>
        <div className={`${baseClass}__field`}>
          <label htmlFor="homebrew-package-name">Homebrew package name</label>
          <input
            id="homebrew-package-name"
            type="text"
            placeholder="e.g. firefox, gifcapture, slack"
            value={packageName}
            onChange={(e) => setPackageName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && packageName.trim() && !isSubmitting) {
                onSubmit();
              }
            }}
            className={`${baseClass}__input`}
          />
        </div>
        {error && <div className={`${baseClass}__error`}>{error}</div>}
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
