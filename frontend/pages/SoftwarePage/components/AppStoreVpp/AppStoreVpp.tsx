import React from "react";
import { InjectedRouter } from "react-router";

const baseClass = "app-store-vpp-form";

interface IAppStoreVppProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
}

const AppStoreVpp = ({ teamId, router, onExit }: IAppStoreVppProps) => {
  return (
    <div className={baseClass}>
      <p>Apple App Store apps purchased via Apple Business Manager.</p>
    </div>
  );
};

export default AppStoreVpp;
