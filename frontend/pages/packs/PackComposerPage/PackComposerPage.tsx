import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";

import { IQuery } from "interfaces/query";
import { ITargetsAPIResponse } from "interfaces/target";
import { IEditPackFormData } from "interfaces/pack";

import { getErrorReason } from "interfaces/errors";
import packsAPI from "services/entities/packs";

import NewPackForm from "components/forms/packs/NewPackForm";
// @ts-ignore
import PackInfoSidePanel from "components/side_panels/PackInfoSidePanel";
import SidePanelPage from "components/SidePanelPage";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";

interface IPackComposerPageProps {
  router: InjectedRouter;
}

const baseClass = "pack-composer";

const PackComposerPage = ({ router }: IPackComposerPageProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [selectedTargetsCount, setSelectedTargetsCount] = useState(0);
  const [isUpdatingPack, setIsUpdatingPack] = useState(false);

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

    setIsUpdatingPack(true);

    try {
      const {
        pack: { id: packID },
      } = await create(formData);
      notify.success("Pack successfully created. Add queries to your pack.");
      router.push(PATHS.PACK(packID));
    } catch (e) {
      if (
        getErrorReason(e, {
          reasonIncludes: "Duplicate entry",
        })
      ) {
        notify.error("Unable to create pack. Pack names must be unique.", {
          response: e,
        });
      } else {
        notify.error("Unable to create pack.", { response: e });
      }
    } finally {
      setIsUpdatingPack(false);
    }
  };

  return (
    <SidePanelPage>
      <>
        <MainContent className={baseClass}>
          <NewPackForm
            className={`${baseClass}__pack-form`}
            handleSubmit={handleSubmit}
            onFetchTargets={onFetchTargets}
            selectedTargetsCount={selectedTargetsCount}
            isPremiumTier={isPremiumTier}
            isUpdatingPack={isUpdatingPack}
          />
        </MainContent>
        <SidePanelContent>
          <PackInfoSidePanel />
        </SidePanelContent>
      </>
    </SidePanelPage>
  );
};

export default PackComposerPage;
