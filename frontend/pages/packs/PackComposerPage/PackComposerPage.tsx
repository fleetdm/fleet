import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { IQuery } from "interfaces/query";
import { ITargetsAPIResponse } from "interfaces/target";
import { IEditPackFormData } from "interfaces/pack";

import { getError } from "services";
import packsAPI from "services/entities/packs"; // @ts-ignore

import PackForm from "components/forms/packs/PackForm"; // @ts-ignore
import PackInfoSidePanel from "components/side_panels/PackInfoSidePanel";

interface IPackComposerPageProps {
  router: InjectedRouter;
}

const baseClass = "pack-composer";

const PackComposerPage = ({ router }: IPackComposerPageProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [selectedTargetsCount, setSelectedTargetsCount] = useState<number>(0);

  const onFetchTargets = (
    query: IQuery,
    targetsResponse: ITargetsAPIResponse
  ) => {
    const { targets_count } = targetsResponse;
    setSelectedTargetsCount(targets_count);
    return false;
  };

  const handleSubmit = async (formData: IEditPackFormData) => {
    const { create } = packsAPI;

    try {
      const {
        pack: { id: packID },
      } = await create(formData);
      router.push(PATHS.PACK(packID));
      renderFlash(
        "success",
        "Pack successfully created. Add queries to your pack."
      );
    } catch (response) {
      const error = getError(response);

      if (error.includes("Error 1062: Duplicate entry")) {
        renderFlash(
          "error",
          "Unable to create pack. Pack names must be unique."
        );
      } else {
        renderFlash("error", "Unable to create pack.");
      }
    }
  };

  return (
    <div className="has-sidebar">
      <PackForm
        className={`${baseClass}__pack-form body-wrap`}
        handleSubmit={handleSubmit}
        onFetchTargets={onFetchTargets}
        selectedTargetsCount={selectedTargetsCount}
        isPremiumTier={isPremiumTier}
      />
      <PackInfoSidePanel />
    </div>
  );
};

export default PackComposerPage;
