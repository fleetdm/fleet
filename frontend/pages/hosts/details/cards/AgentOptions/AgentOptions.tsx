import React from "react";

import { secondsToHms } from "fleet/helpers";

interface IAgentOptionsProps {
  osqueryData: { [key: string]: any };
  wrapFleetHelper: (helperFn: (value: any) => string, value: string) => string;
}

const AgentOptions = ({
  osqueryData,
  wrapFleetHelper,
}: IAgentOptionsProps): JSX.Element => {
  return (
    <div className="section osquery col-50">
      <p className="section__header">Agent options</p>
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">Config TLS refresh</span>
          <span className="info-grid__data">
            {wrapFleetHelper(secondsToHms, osqueryData.config_tls_refresh)}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Logger TLS period</span>
          <span className="info-grid__data">
            {wrapFleetHelper(secondsToHms, osqueryData.logger_tls_period)}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Distributed interval</span>
          <span className="info-grid__data">
            {wrapFleetHelper(secondsToHms, osqueryData.distributed_interval)}
          </span>
        </div>
      </div>
    </div>
  );
};

export default AgentOptions;
