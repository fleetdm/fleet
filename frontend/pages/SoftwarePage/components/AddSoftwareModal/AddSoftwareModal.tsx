import React from "react";
import { InjectedRouter } from "react-router";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";

import AppStoreVpp from "../AppStoreVpp";
import AddPackage from "../AddPackage";

const baseClass = "add-software-modal";

interface IAllTeamsMessageProps {
  onExit: () => void;
}

const AllTeamsMessage = ({ onExit }: IAllTeamsMessageProps) => {
  return (
    <>
      <p>
        Please select a team first. Software can&apos;t be added when{" "}
        <b>All teams</b> is selected.
      </p>
      <div className="modal-cta-wrap">
        <Button variant="brand" onClick={onExit}>
          Done
        </Button>
      </div>
    </>
  );
};

interface IAddSoftwareModalProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
  setAddedSoftwareToken: (token: string) => void;
}

const AddSoftwareModal = ({
  teamId,
  router,
  onExit,
  setAddedSoftwareToken,
}: IAddSoftwareModalProps) => {
  return (
    <Modal
      title="Add software"
      onExit={onExit}
      width="large"
      className={baseClass}
    >
      <>
        {teamId === APP_CONTEXT_ALL_TEAMS_ID ? (
          <AllTeamsMessage onExit={onExit} />
        ) : (
          <TabsWrapper className={`${baseClass}__tabs`}>
            <Tabs>
              <TabList>
                <Tab>Package</Tab>
                <Tab>App Store (VPP)</Tab>
              </TabList>
              <TabPanel>
                <AddPackage
                  teamId={teamId}
                  router={router}
                  onExit={onExit}
                  setAddedSoftwareToken={setAddedSoftwareToken}
                />
              </TabPanel>
              <TabPanel>
                <AppStoreVpp
                  teamId={teamId}
                  router={router}
                  onExit={onExit}
                  setAddedSoftwareToken={setAddedSoftwareToken}
                />
              </TabPanel>
            </Tabs>
          </TabsWrapper>
        )}
      </>
    </Modal>
  );
};

export default AddSoftwareModal;
