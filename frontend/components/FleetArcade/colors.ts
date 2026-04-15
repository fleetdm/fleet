export interface IFleetColor {
  primary: string;
  secondary: string;
  engine: string;
}

// Rotating arcade palette. Each entry supplies a bright primary, a darker
// secondary for fill, and an engine glow. Colors are assigned to fleets in
// the order they appear so fleets always light up in the same hues within
// a session.
const FLEET_PALETTE: readonly IFleetColor[] = [
  { primary: "#00ffff", secondary: "#0088aa", engine: "#66ffff" }, // cyan
  { primary: "#ff00ff", secondary: "#880088", engine: "#ff66ff" }, // magenta
  { primary: "#ffff00", secondary: "#888800", engine: "#ffff66" }, // yellow
  { primary: "#ff8800", secondary: "#884400", engine: "#ffaa44" }, // orange
  { primary: "#39ff14", secondary: "#1a8809", engine: "#66ff44" }, // arcade green
  { primary: "#4488ff", secondary: "#224488", engine: "#66aaff" }, // blue
  { primary: "#ff3366", secondary: "#881133", engine: "#ff6688" }, // pink
  { primary: "#ffffff", secondary: "#666677", engine: "#ccccff" }, // bone white
];

const NO_FLEET_COLOR: IFleetColor = {
  primary: "#888888",
  secondary: "#333344",
  engine: "#aaaaaa",
};

export const NO_FLEET_LABEL = "No fleet";

/**
 * Build a stable fleet-name -> color map from the hosts returned by the API.
 * Fleets are sorted alphabetically (ignoring case) so the assignment is
 * deterministic across refreshes.
 */
export const buildFleetColorMap = (
  fleetNames: readonly string[]
): Record<string, IFleetColor> => {
  const unique = Array.from(
    new Set(fleetNames.map((n) => n || NO_FLEET_LABEL))
  ).filter((n) => n !== NO_FLEET_LABEL);
  unique.sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));

  const map: Record<string, IFleetColor> = {
    [NO_FLEET_LABEL]: NO_FLEET_COLOR,
  };
  unique.forEach((name, idx) => {
    map[name] = FLEET_PALETTE[idx % FLEET_PALETTE.length];
  });
  return map;
};

export const getFleetColor = (
  colorMap: Record<string, IFleetColor>,
  fleetName: string | null | undefined
): IFleetColor => {
  const key = fleetName || NO_FLEET_LABEL;
  return colorMap[key] || NO_FLEET_COLOR;
};
