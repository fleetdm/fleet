import React from "react";
import { useQuery } from "react-query";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

import {
  IVulnHostCountHistogramEntry,
  IVulnHostCountHistogramResponse,
} from "interfaces/vulnerability";
import { getVulnerabilityHostCountHistogram } from "services/entities/vulnerabilities";
import Spinner from "components/Spinner";

import "./_styles.scss";

const baseClass = "vuln-host-count-histogram";

// Mock data for development — remove once backend is deployed.
const MOCK_DATA: IVulnHostCountHistogramEntry[] = [
  { vuln_count_range: "0", hosts_count: 1247, critical: 0, high: 0, medium: 0, low: 0, none: 1247 },
  { vuln_count_range: "1-5", hosts_count: 483, critical: 62, high: 158, medium: 189, low: 74, none: 0 },
  { vuln_count_range: "6-20", hosts_count: 215, critical: 38, high: 87, medium: 64, low: 26, none: 0 },
  { vuln_count_range: "21-50", hosts_count: 89, critical: 24, high: 41, medium: 18, low: 6, none: 0 },
  { vuln_count_range: "51+", hosts_count: 34, critical: 17, high: 12, medium: 4, low: 1, none: 0 },
];

const COLORS: Record<string, string> = {
  critical: "#D66C7B",
  high: "#E8927C",
  medium: "#E2C05A",
  low: "#8EC5C0",
  none: "#C5C7D1",
};

const LABELS: Record<string, string> = {
  critical: "Critical (CVSS 9+)",
  high: "High (CVSS 7-8.9)",
  medium: "Medium (CVSS 4-6.9)",
  low: "Low (CVSS 0.1-3.9)",
  none: "No known CVSS",
};

interface IProps {
  teamId?: number;
  isSoftwareEnabled: boolean;
}

interface ITooltipProps {
  active?: boolean;
  payload?: Array<{ name: string; value: number; color: string }>;
  label?: string;
}

const CustomTooltip = ({ active, payload, label }: ITooltipProps) => {
  if (!active || !payload) return null;
  const total = payload.reduce((sum, e) => sum + e.value, 0);
  return (
    <div style={{ background: "#192147", borderRadius: 6, padding: "12px 16px", color: "#fff", fontSize: 13, boxShadow: "0 4px 12px rgba(0,0,0,0.15)" }}>
      <p style={{ margin: "0 0 8px", fontWeight: 600 }}>
        {label === "0" ? "0 vulnerabilities" : `${label} vulnerabilities`}
      </p>
      <p style={{ margin: "0 0 6px", fontSize: 12, opacity: 0.8 }}>
        {total.toLocaleString()} hosts total
      </p>
      {payload.filter((e) => e.value > 0).reverse().map((e) => (
        <div key={e.name} style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4 }}>
          <div style={{ width: 8, height: 8, borderRadius: 2, background: e.color }} />
          <span>{LABELS[e.name] || e.name}: {e.value.toLocaleString()}</span>
        </div>
      ))}
    </div>
  );
};

const VulnHostCountHistogram = ({ teamId, isSoftwareEnabled }: IProps) => {
  const { data, isLoading } = useQuery<IVulnHostCountHistogramResponse>(
    [{ scope: "vuln-host-count-histogram", teamId }],
    () => getVulnerabilityHostCountHistogram(teamId),
    { enabled: isSoftwareEnabled, keepPreviousData: true, retry: false }
  );

  const chartData = data?.histogram ?? MOCK_DATA;

  if (isLoading) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__loading`}><Spinner /></div>
      </div>
    );
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <div>
          <h3 className={`${baseClass}__title`}>Vulnerability exposure by host</h3>
          <p className={`${baseClass}__subtitle`}>
            Hosts grouped by total vulnerability count, colored by highest severity
          </p>
        </div>
      </div>
      <div className={`${baseClass}__chart-container`}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={chartData} margin={{ top: 8, right: 24, left: 8, bottom: 8 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#E2E4EA" vertical={false} />
            <XAxis
              dataKey="vuln_count_range"
              tick={{ fontSize: 13, fill: "#515774" }}
              tickLine={false}
              axisLine={{ stroke: "#E2E4EA" }}
              label={{ value: "Number of vulnerabilities", position: "insideBottom", offset: -2, style: { fontSize: 12, fill: "#8B8FA2" } }}
            />
            <YAxis
              tick={{ fontSize: 13, fill: "#515774" }}
              tickLine={false}
              axisLine={false}
              tickFormatter={(v: number) => (v >= 1000 ? `${(v / 1000).toFixed(1)}k` : String(v))}
              label={{ value: "Number of hosts", angle: -90, position: "insideLeft", offset: 10, style: { fontSize: 12, fill: "#8B8FA2", textAnchor: "middle" } }}
            />
            <Tooltip content={<CustomTooltip />} cursor={{ fill: "rgba(25,33,71,0.04)" }} />
            <Bar dataKey="none" stackId="s" fill={COLORS.none} />
            <Bar dataKey="low" stackId="s" fill={COLORS.low} />
            <Bar dataKey="medium" stackId="s" fill={COLORS.medium} />
            <Bar dataKey="high" stackId="s" fill={COLORS.high} />
            <Bar dataKey="critical" stackId="s" fill={COLORS.critical} radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div className={`${baseClass}__legend`}>
        {Object.entries(COLORS).reverse().map(([key, color]) => (
          <div key={key} className={`${baseClass}__legend-item`}>
            <div className={`${baseClass}__legend-dot`} style={{ backgroundColor: color }} />
            {LABELS[key]}
          </div>
        ))}
      </div>
    </div>
  );
};

export default VulnHostCountHistogram;
