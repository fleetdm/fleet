import React from "react";

import { secondsToHms } from "utilities/helpers";

interface IAgentOptionsProps {
  osqueryData: { [key: string]: any };
  wrapFleetHelper: (helperFn: (value: any) => string, value: string) => string;
  platform?: string;
}

// TODO: confirm these values
const CHROMEOS_AGENT_OPTIONS = ["10 secs", "10 secs", "10 secs"];

const AgentOptions = ({
  osqueryData,
  wrapFleetHelper,
  platform,
}: IAgentOptionsProps): JSX.Element => {
  let configTLSRefresh;
  let loggerTLSPeriod;
  let distributedInterval;

  // TODO: check this is the correct string to expect
  if (platform === "chromeos") {
    [
      configTLSRefresh,
      loggerTLSPeriod,
      distributedInterval,
    ] = CHROMEOS_AGENT_OPTIONS;
  } else {
    configTLSRefresh = wrapFleetHelper(
      secondsToHms,
      osqueryData.config_tls_refresh
    );
    loggerTLSPeriod = wrapFleetHelper(
      secondsToHms,
      osqueryData.logger_tls_period
    );
    distributedInterval = wrapFleetHelper(
      secondsToHms,
      osqueryData.distributed_interval
    );
  }
  return (
    <div className="section osquery col-50">
      <p className="section__header">Agent options</p>
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">Config TLS refresh</span>
          <span className="info-grid__data">{configTLSRefresh}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Logger TLS period</span>
          <span className="info-grid__data">{loggerTLSPeriod}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Distributed interval</span>
          <span className="info-grid__data">{distributedInterval}</span>
        </div>
      </div>
    </div>
  );
};

export default AgentOptions;
