import React, { useCallback, useContext, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "react-query";

import { NotificationContext } from "context/notification";
import { IBaselineManifest } from "interfaces/baseline";

import baselinesAPI from "services/entities/baselines";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "baselines";

export type IBaselinesProps = IOSSettingsCommonProps;

const Baselines = ({
  currentTeamId,
  onMutation,
}: IBaselinesProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const queryClient = useQueryClient();
  const [applyingId, setApplyingId] = useState<string | null>(null);

  const {
    data: baselinesData,
    isLoading,
    isError,
  } = useQuery(["baselines"], () => baselinesAPI.loadAll(), {
    refetchOnWindowFocus: false,
    retry: false,
  });

  const applyMutation = useMutation(
    (baselineId: string) =>
      baselinesAPI.apply({
        baseline_id: baselineId,
        team_id: currentTeamId,
      }),
    {
      onSuccess: (data) => {
        const total =
          (data.profiles_created?.length ?? 0) +
          (data.policies_created?.length ?? 0) +
          (data.scripts_created?.length ?? 0);
        renderFlash(
          "success",
          `Baseline applied: ${total} resources created.`
        );
        queryClient.invalidateQueries(["baselines"]);
        onMutation();
      },
      onError: () => {
        renderFlash("error", "Failed to apply baseline. Please try again.");
      },
      onSettled: () => {
        setApplyingId(null);
      },
    }
  );

  const removeMutation = useMutation(
    (baselineId: string) => baselinesAPI.remove(baselineId, currentTeamId),
    {
      onSuccess: () => {
        renderFlash("success", "Baseline removed successfully.");
        queryClient.invalidateQueries(["baselines"]);
        onMutation();
      },
      onError: () => {
        renderFlash("error", "Failed to remove baseline. Please try again.");
      },
    }
  );

  const handleApply = useCallback(
    (baselineId: string) => {
      setApplyingId(baselineId);
      applyMutation.mutate(baselineId);
    },
    [applyMutation]
  );

  const handleRemove = useCallback(
    (baselineId: string) => {
      removeMutation.mutate(baselineId);
    },
    [removeMutation]
  );

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return (
      <div className={baseClass}>
        <SectionHeader title="Security baselines" />
        <p>Failed to load baselines.</p>
      </div>
    );
  }

  const baselines = baselinesData?.baselines ?? [];

  const renderBaseline = (baseline: IBaselineManifest) => {
    const isApplying = applyingId === baseline.id;
    const categoryCount = baseline.categories.length;
    const profileCount = baseline.categories.reduce(
      (sum, c) => sum + c.profiles.length,
      0
    );
    const policyCount = baseline.categories.reduce(
      (sum, c) => sum + c.policies.length,
      0
    );

    return (
      <div key={baseline.id} className={`${baseClass}__card`}>
        <div className={`${baseClass}__card-header`}>
          <h3>{baseline.name}</h3>
          <span className={`${baseClass}__version`}>v{baseline.version}</span>
        </div>
        <p className={`${baseClass}__description`}>{baseline.description}</p>
        <div className={`${baseClass}__stats`}>
          <span>{categoryCount} categories</span>
          <span>{profileCount} profiles</span>
          <span>{policyCount} policies</span>
        </div>
        <div className={`${baseClass}__actions`}>
          <Button
            variant="default"
            onClick={() => handleApply(baseline.id)}
            isLoading={isApplying}
            disabled={currentTeamId === 0}
          >
            Apply to team
          </Button>
          <Button
            variant="text-link"
            onClick={() => handleRemove(baseline.id)}
            disabled={currentTeamId === 0}
          >
            Remove
          </Button>
        </div>
        {currentTeamId === 0 && (
          <p className={`${baseClass}__team-required`}>
            Select a team to apply or remove this baseline.
          </p>
        )}
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Security baselines" />
      <PageDescription
        content="Security baselines are curated bundles of MDM profiles, verification
        policies, and remediation scripts. Applying a baseline to a team creates
        these resources using Fleet's existing profile, policy, and script systems."
      />
      <div className={`${baseClass}__list`}>
        {baselines.map(renderBaseline)}
      </div>
    </div>
  );
};

export default Baselines;
